package main

import (
    "context"
    "crypto/tls"
    "fmt"
    "net/url"
    "os"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
    "golang.org/x/crypto/bcrypt"

    "clipboard-remote/clipboard"
    "clipboard-remote/utils"
)

type Client struct {
    sync.Mutex

    client *utils.ClientConfig
    ID     string
    conn   *websocket.Conn

    // {string: chan *types.WebsocketMessage}
    // readChs sync.Map
    writeCh chan *utils.WebsocketMessage
}

// NewClient creates a new ws client
func NewClient(c *utils.ClientConfig) *Client {
    id, err := os.Hostname()
    if err != nil {
        id = uuid.NewString()
        if err != nil {
            panic(fmt.Errorf("failed to initialize daemon: %v", err))
        }
    }
    return &Client{
        ID:      id,
        writeCh: make(chan *utils.WebsocketMessage, 10),
        client:  c,
    }
}

func (c *Client) connect() error {
    c.Lock()
    defer c.Unlock()

    dial := websocket.Dialer{TLSClientConfig: &tls.Config{
        InsecureSkipVerify: c.client.InsecureSkipVerify,
    }}

    u := url.URL{Scheme: "wss", Host: c.client.Host, Path: c.client.WebsocketPath}
    conn, _, err := dial.Dial(u.String(), nil)
    if err != nil {
        return fmt.Errorf("failed to dial(%s): %w", u.String(), err)
    }
    c.conn = conn

    // hash password
    hashBytes, err := bcrypt.GenerateFromPassword([]byte(c.client.Auth.Password), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    // handshake with server
    c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
    creds := c.client.Auth.User + ":" + utils.BytesToString(hashBytes)
    err = c.conn.WriteMessage(websocket.BinaryMessage, (&utils.WebsocketMessage{
        Action: utils.ActionHandshakeRegister,
        UserID: c.ID,
        Data:   utils.StringToBytes(creds),
    }).Encode())

    if err != nil {
        return fmt.Errorf("failed to send handshake message: %w", err)
    }

    c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
    _, msg, err := c.conn.ReadMessage()
    if err != nil {
        return fmt.Errorf("failed to read message for handshake: %w", err)
    }

    wsm := &utils.WebsocketMessage{}
    err = wsm.Decode(msg)
    if err != nil {
        return fmt.Errorf("failed to handshake with server: %w", err)
    }

    switch wsm.Action {
    case utils.ActionHandshakeReady:
        log.Infoln("Hand shake succeed:", c.ID)
    default:
        // close the connection if handshake is not ready
        c.conn.Close()
        return fmt.Errorf("failed to handshake with server: %w", err)
    }

    return nil
}

// wsReconnect tries to reconnect to the server and returns
// until it connects to the server.
func (c *Client) reconnect(ctx context.Context) {
    tk := time.NewTicker(10 * time.Second)
    defer tk.Stop()
    for {
        select {
        case <-ctx.Done():
            return
        case <-tk.C:
            err := c.connect()
            if err == nil {
                log.Infoln("Connected to server succeed.")
                return
            }
            log.Errorf("%v\n", err)
            log.Infoln("Retry in 10 seconds..")
        }
    }
}

func (c *Client) handleIO(ctx context.Context, clipData <-chan []byte) {
    if c.conn == nil {
        c.reconnect(ctx)
    }

    log.Debugln("Client id:", c.ID)

    // when auto mode, watch the clipboard content
    if c.client.Mode == "auto" {
        go c.watchClipboard(ctx, clipData)
    }

    go c.readFromServer(ctx)
    go c.writeToServer(ctx)
}

func (c *Client) readFromServer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            log.Infoln("Exit read routine.")
            return
        default:
            c.conn.SetReadDeadline(time.Time{})
            _, msg, err := c.conn.ReadMessage()
            if err != nil {
                log.Errorf("Failed to read message from server: %v", err)

                c.Lock()
                c.conn.Close()
                c.conn = nil
                c.Unlock()

                // block until connection is ready again
                c.reconnect(ctx)
                continue
            }

            wsm := &utils.WebsocketMessage{}
            err = wsm.Decode(msg)
            if err != nil {
                log.Errorf("Failed to read message: %v", err)
                continue
            }

            switch wsm.Action {
            case utils.ActionClipboardChanged:
                log.Debugf("Clipboard data has changed from %s, sync with local...", wsm.UserID)
                clipboard.Write(wsm.Data)
                log.Debugf("Clipboard data has changed from %s, sync succeed.", wsm.UserID)
            }
        }
    }
}

func (c *Client) writeToServer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            log.Infoln("Exit write routine.")
            return
        case msg := <-c.writeCh:
            if c.conn == nil {
                log.Errorln("connection is not ready yet for user:", c.ID)
                continue
            }

            c.conn.SetWriteDeadline(time.Time{})
            err := c.conn.WriteMessage(websocket.BinaryMessage, msg.Encode())
            if err != nil {
                log.Errorf("failed to write message to server: %v", err)
                return
            }
        }
    }
}

func (c *Client) watchClipboard(ctx context.Context, clipData <-chan []byte) {
    for {
        select {
        case <-ctx.Done():
            log.Infoln("Exit watch routine.")
            return
        case data, ok := <-clipData:
            if c.conn == nil || !ok {
                log.Errorln("connection is not ready yet for user:", c.ID)
                continue
            }

            if !ok {
                log.Errorln("Clipboard data channel has been closed.")
                continue
            }

            c.writeCh <- &utils.WebsocketMessage{
                Action: utils.ActionClipboardChanged,
                UserID: c.ID,
                Data:   data,
            }
        }
    }
}

func (c *Client) close() {
    if c.conn == nil {
        return
    }

    _ = c.conn.WriteMessage(websocket.BinaryMessage, (&utils.WebsocketMessage{
        Action: utils.ActionTerminate,
        UserID: c.ID,
    }).Encode())

    h := c.conn.CloseHandler()
    h(websocket.CloseNormalClosure, "")

    c.conn.Close()
}
