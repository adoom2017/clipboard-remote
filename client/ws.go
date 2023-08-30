package main

import (
    "context"
    "crypto/tls"
    "encoding/base64"
    "fmt"
    "net/http"
    "net/url"
    "os"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
    "golang.org/x/crypto/bcrypt"

    "clipboard-remote/clipboard"
    util "clipboard-remote/common"
)

type Client struct {
    sync.Mutex

    client *util.ClientConfig
    ID     string
    conn   *websocket.Conn

    // {string: chan *types.WebsocketMessage}
    readChs sync.Map
    writeCh chan *util.WebsocketMessage
}

// NewClient creates a new ws client
func NewClient(c *util.ClientConfig) *Client {
    id, err := os.Hostname()
    if err != nil {
        id = uuid.NewString()
        if err != nil {
            panic(fmt.Errorf("failed to initialize daemon: %v", err))
        }
    }
    return &Client{
        ID:      id,
        writeCh: make(chan *util.WebsocketMessage, 10),
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

    hashBytes, err := bcrypt.GenerateFromPassword([]byte(c.client.Auth.Password), bcrypt.DefaultCost)
    if err != nil {
        return fmt.Errorf("failed to hash password: %w", err)
    }

    creds := c.client.Auth.User + ":" + util.BytesToString(hashBytes)
    token := base64.StdEncoding.EncodeToString(util.StringToBytes(creds))

    reqHeader := make(http.Header)
    reqHeader.Add("ID", c.ID)
    reqHeader.Add("Authorization", token)

    c.conn, _, err = dial.Dial(u.String(), reqHeader)
    if err != nil {
        return fmt.Errorf("failed to dial(%s): %w", u.String(), err)
    }

    log.Infof("Succeed connecting to %s, ID: %s.", u.String(), c.ID)

    // handshake with midgard server
    err = c.conn.WriteMessage(websocket.BinaryMessage, (&util.WebsocketMessage{
        Action: util.ActionHandshakeRegister,
        UserID: c.ID,
        Data:   nil,
    }).Encode())

    if err != nil {
        return fmt.Errorf("failed to send handshake message: %w", err)
    }

    _, msg, err := c.conn.ReadMessage()
    if err != nil {
        return fmt.Errorf("failed to read message for handshake: %w", err)
    }

    wsm := &util.WebsocketMessage{}
    err = wsm.Decode(msg)
    if err != nil {
        return fmt.Errorf("failed to handshake with server: %w", err)
    }

    switch wsm.Action {
    case util.ActionHandshakeReady:
        if wsm.UserID != c.ID {
            c.ID = wsm.UserID
            log.Infoln("conflict hostname, updated daemon id: ", c.ID)
        }
    default:
        // close the connection if handshake is not ready
        c.conn.Close()
        return fmt.Errorf("failed to handshake with server: %w", err)
    }

    return nil
}

// wsReconnect tries to reconnect to the midgard server and returns
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
                log.Infoln("connected to server succeed.")
                return
            }
            log.Errorf("%v\n", err)
            log.Infoln("retry in 10 seconds..")
        }
    }
}

func (c *Client) handleIO(ctx context.Context) {
    if c.conn == nil {
        c.reconnect(ctx)
    }

    log.Infoln("Client id:", c.ID)
    go c.readFromServer(ctx)
    c.writeToServer(ctx)
    c.close()
}

func (c *Client) readFromServer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        default:
            _, msg, err := c.conn.ReadMessage()
            if err != nil {
                log.Errorf("Failed to read message from server: %v", err)

                c.Lock()
                c.conn = nil
                c.Unlock()

                // block until connection is ready again
                c.reconnect(ctx)
                continue
            }

            wsm := &util.WebsocketMessage{}
            err = wsm.Decode(msg)
            if err != nil {
                log.Printf("failed to read message: %v", err)
                continue
            }

            // duplicate messages to all readers, readers should not edit the message
            c.readChs.Range(func(k, v interface{}) bool {
                readerCh := v.(chan *util.WebsocketMessage)
                readerCh <- wsm
                return true
            })

            switch wsm.Action {
            case util.ActionClipboardChanged:
                log.Infof("Clipboard data has changed from %s, sync with local...", wsm.UserID)
                clipboard.Write(wsm.Data)
                log.Infof("Clipboard data has changed from %s, sync succeed.", wsm.UserID)
            }
        }
    }
}

func (c *Client) writeToServer(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            if c.conn == nil {
                log.Errorln("Connection was not ready for user:", c.ID)
            }
            return
        case msg := <-c.writeCh:
            if c.conn == nil {
                log.Errorln("connection is not ready yet for user:", c.ID)
                continue
            }

            err := c.conn.WriteMessage(websocket.BinaryMessage, msg.Encode())
            if err != nil {
                log.Errorf("failed to write message to server: %v", err)
                return
            }
        }
    }
}

func (c *Client) close() {
    _ = c.conn.WriteMessage(websocket.BinaryMessage, (&util.WebsocketMessage{
        Action: util.ActionTerminate,
        UserID: c.ID,
    }).Encode())

    h := c.conn.CloseHandler()
    h(websocket.CloseNormalClosure, "")

    c.conn.Close()
}
