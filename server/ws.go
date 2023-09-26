package main

import (
    util "clipboard-remote/common"
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

    // Maximum message size allowed from peer.
    maxMessageSize = 10 * 1024 * 1024

    // Time allowed to read the next pong message from the peer.
    pongWait = 60 * time.Second

    // Send pings to peer with this period. Must be less than pongWait.
    pingPeriod = (pongWait * 9) / 10

    // Time to wait before force close on connection.
    //closeGracePeriod = 10 * time.Second
)

// Server is a middleman between the websocket connection and the router.
type Server struct {
    router *Router

    // The websocket connection.
    conn *websocket.Conn

    // Buffered channel of outbound messages.
    send chan []byte

    // client identify
    id string

    // message username
    username string
}

// readMsgFromWs read messages from the websocket connection to the router.
func (c *Server) readMsgFromWs() {
    // clean func
    defer func() {
        c.router.unregister <- c
        c.conn.Close()
    }()

    // set pong message handler
    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

    for {
        _, msg, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Errorf("error: %v", err)
            }
            break
        }

        // handle the client message
        wsm := &util.WebsocketMessage{}
        err = wsm.Decode(msg)
        if err != nil {
            log.Errorf("Error messaage: %v", err)
            continue
        }

        switch wsm.Action {
        case util.ActionHandshakeRegister:
            user, ok := auth(wsm.Data)
            if !ok {
                log.Errorln("Failed to auth user:", user)
                close(c.send)
                return
            } else {
                c.router.clients[c] = user
                wsm := &util.WebsocketMessage{
                    Action: util.ActionHandshakeReady,
                    UserID: wsm.UserID,
                    Data:   nil,
                }
                c.send <- wsm.Encode()

                c.id = wsm.UserID
                c.username = user

                log.Infof("User: %s login succeed.", user)
            }
        case util.ActionClipboardChanged:
            // broadcast clip content to user's all client
            DB.InsertClipContent(&util.ClipContentInfo{ClientID: c.id, Username: c.username, Content: base64.StdEncoding.EncodeToString(wsm.Data)})
            c.router.broadcast <- &Message{id: c.id, username: c.username, content: wsm.Data}

            content := DB.GetClipContentByID(c.id)
            temp, _ := base64.StdEncoding.DecodeString(content)

            aaa, _ := util.DecodeToStruct(temp)
            log.Infof("Type: %d, Name: %s, Buff: %s.", aaa.Type, aaa.Name, string(aaa.Buff))

        default:
            // close the connection if handshake is not ready
            c.conn.Close()
            return
        }

    }
}

// writeMsgToWs pumps messages from the hub to the websocket connection.
func (c *Server) writeMsgToWs() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
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
            log.Infoln("tiker timeout, send ping message")
        }
    }
}

func auth(token []byte) (string, bool) {
    tokens := strings.Split(util.BytesToString(token), ":")
    if len(tokens) != 2 {
        log.Errorln("Invalid token:", token)
        return "", false
    }
    user := tokens[0]
    pass := DB.GetPassword(user)

    err := bcrypt.CompareHashAndPassword(util.StringToBytes((tokens[1])), util.StringToBytes(pass))
    if err != nil {
        log.Errorf("Failed to auth user(%s), error: %v.", user, err)
        return user, false
    }

    return user, true
}

// ServeWs handles websocket requests from the peer.
func ServeWs(router *Router, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

    client := &Server{
        router: router,
        conn:   conn,
        send:   make(chan []byte, 256),
    }

    // Allow collection of memory referenced by the caller by doing all work in
    // new goroutines.
    go client.writeMsgToWs()
    go client.readMsgFromWs()
}
