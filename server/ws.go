package main

import (
  "clipboard-remote/utils"
  "encoding/base64"
  "net/http"
  "strings"
  "time"

  "github.com/gorilla/websocket"
  log "github.com/sirupsen/logrus"
  "golang.org/x/crypto/bcrypt"
)

const (
  // Time allowed to write a message to the peer.
  writeWait = 10 * time.Second

  // Time allowed to read the next pong message from the peer.
  pongWait = 60 * time.Second

  // Send pings to peer with this period. Must be less than pongWait.
  pingPeriod = (pongWait * 9) / 10

  // Time to wait before force close on connection.
  //closeGracePeriod = 10 * time.Second
)

// Client is a middleman between the websocket connection and the router.
type Client struct {
  router *Router

  // The websocket connection.
  conn *websocket.Conn

  // Buffered channel of outbound messages.
  send chan []byte

  // client identify
  id string

  // message username
  username string

  // is clipboard content change auto send
  auto bool
}

// handRegisterMsg register handle function
func (c *Client) handRegisterMsg(wsm *utils.WebsocketMessage) error {
  user, mode, ok := authWS(wsm.Data)
  if !ok {
    return utils.ErrAuthFailed
  }

  // reply ready message to client
  shakeReadyMsg := &utils.WebsocketMessage{
    Action: utils.ActionHandshakeReady,
    UserID: wsm.UserID,
    Data:   nil,
  }
  c.send <- shakeReadyMsg.Encode()

  c.id = wsm.UserID
  c.username = user

  if mode == "auto" {
    c.auto = true
  } else {
    c.auto = false
  }

  // register client to router
  c.router.register <- c

  return nil
}

// handClipboardContentMsg clipboard content change handle function
func (c *Client) handClipboardContentMsg(wsm *utils.WebsocketMessage) error {
  if c.id == "" || c.username == "" {
    return utils.ErrUnAuthenticatedClient
  }

  // insert clipboard data into database
  err := DB.InsertClipContent(&utils.ClipContentInfo{
    ClientID: c.id,
    Username: c.username,
    Content:  base64.StdEncoding.EncodeToString(wsm.Data),
  })

  if err != nil {
    log.Errorf("Failed to insert clipcontent to database, id: %s, user: %s.", c.id, c.username)
    // Ignore errors; therefore, no return
  }

  // broadcast clipboard content to user's all client
  c.router.broadcast <- &Message{
    id:       c.id,
    username: c.username,
    content:  wsm.Data,
  }

  return nil
}

// readMsgFromWs read messages from the websocket connection to the router.
func (c *Client) readMsgFromWs() {
  // clean func
  // clear function
  defer func() {
    c.router.unregister <- c
  }()

  // set pong message handler
  c.conn.SetReadLimit(int64(GlobalConfig.MaxMsgSize))
  c.conn.SetReadDeadline(time.Now().Add(pongWait))
  c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

  for {
    _, msg, err := c.conn.ReadMessage()
    if err != nil {
      if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
        log.Errorln("Read message from websocket, error:", err)
      }
      return
    }

    // handle the client message
    wsm := &utils.WebsocketMessage{}
    err = wsm.Decode(msg)
    if err != nil {
      log.Errorf("Error message: %v.", err)
      continue
    }

    switch wsm.Action {
    case utils.ActionHandshakeRegister:
      err = c.handRegisterMsg(wsm)
      if err != nil {
        log.Errorf("Failed to handle register message from client: %s, error: %v.", wsm.UserID, err)
        return
      }
      log.Infoln("Client register succeed:", wsm.UserID)
    case utils.ActionClipboardChanged:
      err = c.handClipboardContentMsg(wsm)
      if err != nil {
        log.Errorf("Failed to handle clipboard message from client: %s, error: %v.", wsm.UserID, err)
        return
      }
      log.Infoln("Client clipboard info change:", wsm.UserID)
    case utils.ActionTerminate:
      // client unregister
      log.Infoln("Client terminate:", wsm.UserID)
      return
    }
  }
}

// writeMsgToWs pumps messages from the hub to the websocket connection.
func (c *Client) writeMsgToWs() {
  ticker := time.NewTicker(pingPeriod)

  // clear function
  defer func() {
    // stop the timer
    ticker.Stop()

    // quit then close the connection
    c.conn.Close()
  }()

  for {
    select {
    // write data to websocket from send buffer
    case message, ok := <-c.send:
      c.conn.SetWriteDeadline(time.Now().Add(writeWait))
      if !ok {
        // the channel has been closed.
        c.conn.WriteMessage(websocket.CloseMessage, []byte{})
        return
      }

      w, err := c.conn.NextWriter(websocket.TextMessage)
      if err != nil {
        return
      }
      w.Write(message)

      // Add queued chat messages to the current websocket message.
      n := len(c.send)
      for i := 0; i < n; i++ {
        w.Write(<-c.send)
      }

      if err := w.Close(); err != nil {
        return
      }
    case <-ticker.C:
      c.conn.SetWriteDeadline(time.Now().Add(writeWait))
      if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
        return
      }
      log.Debugln("tiker timeout, send ping message.")
    }
  }
}

// authWS client authentication, return username, mode, succeed
func authWS(token []byte) (string, string, bool) {
  tokens := strings.Split(utils.BytesToString(token), ":")
  if len(tokens) != 3 {
    log.Errorln("Invalid token:", token)
    return "", "", false
  }
  user := tokens[0]
  pass := DB.GetPassword(user)

  err := bcrypt.CompareHashAndPassword(utils.StringToBytes((tokens[1])), utils.StringToBytes(pass))
  if err != nil {
    log.Errorf("Failed to auth user(%s), error: %v.", user, err)
    return user, tokens[2], false
  }

  return user, tokens[2], true
}

// ServeWs handles websocket requests from the peer.
func ServeWs(router *Router, w http.ResponseWriter, r *http.Request) {
  conn, err := upgrader.Upgrade(w, r, nil)
  if err != nil {
    log.Errorln("Failed to accept wss socket:", err)
    return
  }

  client := &Client{
    router: router,
    conn:   conn,
    send:   make(chan []byte, 256),
  }

  // Allow collection of memory referenced by the caller by doing all work in
  // new goroutines.
  go client.writeMsgToWs()
  go client.readMsgFromWs()
}
