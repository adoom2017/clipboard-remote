package main

import (
    "context"
    "crypto/tls"
    "flag"
    "net"
    "net/http"
    "net/url"
    "os"
    "os/signal"
    "time"

    "clipboard-remote/clipboard"
    util "clipboard-remote/common"

    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
    "golang.org/x/crypto/bcrypt"
)

var (
    configPath = flag.String("config", "client.yaml", "client config file path")
)

func init() {
    // Set the log output to stdout
    log.SetReportCaller(true)
    log.SetFormatter(&util.Formatter{
        HideKeys:    true,
        CallerFirst: true,
        NoColors:    true,
    })

    // Set the log output to the specified file
    log.SetOutput(os.Stdout)

    // Set the log level
    log.SetLevel(log.InfoLevel)
}

func main() {
    flag.Parse()

    clientConfig, err := util.ClientConfigRead(*configPath)
    if err != nil {
        log.Errorf("Failed to load client config file(%s), err: %v.", *configPath, err)
        return
    }

    // add interrupt sigal
    interrupt := make(chan os.Signal, 1)
    signal.Notify(interrupt, os.Interrupt)

    tlsConfig := tls.Config{
        InsecureSkipVerify: clientConfig.InsecureSkipVerify,
    }
    dial := websocket.Dialer{TLSClientConfig: &tlsConfig}

    u := url.URL{Scheme: "wss", Host: clientConfig.Host, Path: clientConfig.WebsocketPath}
    // log.Infof("Succeed connecting to %s, uuid: %s.", u.String(), clientConfig.Uuid)

    reqHeader := make(http.Header)
    //reqHeader.Add("ID", clientConfig.Uuid)
    reqHeader.Add("User-Name", clientConfig.Auth.User)

    hashBytes, err := bcrypt.GenerateFromPassword([]byte(clientConfig.Auth.Password), bcrypt.DefaultCost)
    if err != nil {
        log.Errorln("Failed to hash password.")
        return
    }

    reqHeader.Add("Token", string(hashBytes))

    c, _, err := dial.Dial(u.String(), reqHeader)
    if err != nil {
        log.Fatalln(err)
    }
    defer c.Close()

    done := make(chan struct{})

    // ping handler for keepalive
    c.SetPingHandler(func(message string) error {
        // log.Infoln("Receive ping message from server:", c.RemoteAddr().String())
        err := c.WriteControl(websocket.PongMessage, []byte(message), time.Now().Add(10*time.Second))
        if err == websocket.ErrCloseSent {
            return nil
        } else if _, ok := err.(net.Error); ok {
            return nil
        }
        return err
    })

    // receive clipboard content from server
    // then add the content to local clipboard
    go func() {
        defer close(done)
        for {
            _, message, err := c.ReadMessage()
            if err != nil {
                log.Errorln("Read error:", err)
                return
            }

            clipInfo, err := util.DecodeToStruct(message)
            if err != nil {
                log.Errorln("Unsupported message:", err)
                continue
            }

            switch clipInfo.Type {
            case util.CLIP_TEXT:
                log.Infof("Recv Text Message: %s", string(clipInfo.Buff))
                clipboard.Write(message)
                log.Infof("Set Text Message: %s to clipboard.", string(clipInfo.Buff))
            case util.CLIP_PATH:
                log.Infoln("Recv Path Message.")
            default:
                log.Errorln("Unsupported format:", clipInfo.Type)
            }
        }
    }()

    // watch context, used for watch break
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    dataCh := clipboard.Watch(ctx)

    for {
        select {
        case <-done:
            log.Infoln("Bye bye!")
            return
        case data, ok := <-dataCh:
            if !ok {
                log.Errorln("Clipboard data channel has been closed.")
                return
            }

            c.SetWriteDeadline(time.Now().Add(10 * time.Second))
            err := c.WriteMessage(websocket.BinaryMessage, data)
            if err != nil {
                log.Errorln("Failed to write message:", err)
                return
            }
            log.Infoln("Send clipboard message succeed.")

        case <-interrupt:
            log.Infoln("Interrupt manually by user.")

            // Cleanly close the connection by sending a close message and then
            // waiting (with timeout) for the server to close the connection.
            err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
            if err != nil {
                log.Errorln("Failed to send close message:", err)
                return
            }
        }
    }
}
