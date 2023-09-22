package main

import (
    "encoding/base64"
    "net/http"
    "strings"
    "time"

    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
    "golang.org/x/crypto/bcrypt"

    util "clipboard-remote/common"
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

    // username
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
        _, content, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Errorf("error: %v", err)
            }
            break
        }

        c.router.broadcast <- &Message{id: c.id, content: content}
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

// ServeWs handles websocket requests from the peer.
func ServeWs(router *Router, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println(err)
        return
    }

    token, err := base64.StdEncoding.DecodeString(r.Header.Get("Authorization"))
    if err != nil {
        log.Errorln("Failed to decode token:", token, err)
        return
    }

    tokens := strings.Split(util.BytesToString(token), ":")
    pass := DB.GetPassword(tokens[0])

    err = bcrypt.CompareHashAndPassword(util.StringToBytes((tokens[1])), util.StringToBytes(pass))
    if err != nil {
        log.Errorf("Failed to auth user(%s), error: %v.", tokens[0], err)
        return
    }

    client := &Server{router: router, conn: conn, send: make(chan []byte, 256), id: r.Header.Get("ID"), username: tokens[0]}
    client.router.register <- client

    // Allow collection of memory referenced by the caller by doing all work in
    // new goroutines.
    go client.writeMsgToWs()
    go client.readMsgFromWs()
}
