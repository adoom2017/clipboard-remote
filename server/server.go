package main

import (
    "clipboard-remote/utils"
    "context"
    "encoding/base64"
    "flag"
    "net/http"
    "os"
    "os/signal"
    "path"
    "syscall"
    "time"

    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
)

var (
    configDir  = flag.String("d", "config", "server config directory")
    configFile = flag.String("f", "server.yaml", "server config filename")

    upgrader = websocket.Upgrader{
        ReadBufferSize:    4096,
        WriteBufferSize:   4096,
        EnableCompression: true,
    }

    DB *utils.DBInfo
)

func init() {
    // Set the report callers to true
    log.SetReportCaller(true)
    // Set the formatter to include the function name and line number
    log.SetFormatter(&utils.Formatter{
        HideKeys:    true,
        CallerFirst: true,
        NoColors:    true,
    })

    // Set the output to the standard output
    log.SetOutput(os.Stdout)

    // Set the log level
    log.SetLevel(log.InfoLevel)
}

func getClipboardHandler(w http.ResponseWriter, r *http.Request) {
    user := r.URL.Query().Get("username")
    buff, err := base64.StdEncoding.DecodeString(DB.GetClipContentByName(user))
    if err != nil {
        log.Errorln("Failed to get clipboard content for user:", user, err)
        w.Write([]byte("Get clipboard info failed."))
        return
    }

    content, err := utils.DecodeToStruct(buff)
    if err != nil {
        log.Errorln("Failed to get clipboard content for user:", user, err)
        w.Write([]byte("Get clipboard info failed."))
        return
    }

    if content.Type != utils.CLIP_TEXT {
        w.Write([]byte("Unsupported content type."))
        return
    }

    w.Write(content.Buff)
}

func main() {
    // Parse command line arguments
    flag.Parse()

    configPath := path.Join(*configDir, *configFile)
    serverConfig, err := utils.ServerConfigRead(configPath)
    if err != nil {
        log.Errorf("Failed to load server config file(%s), err: %v.", configPath, err)
        return
    }

    // Init sqlite database
    dbFilePath := path.Join(*configDir, "server.sqlite3")
    if utils.Exists(dbFilePath) {
        DB = utils.InitDB(dbFilePath)
    } else {
        DB = utils.InitDB(dbFilePath)
        err = DB.CreateUserInfoTable()
        if err != nil {
            log.Errorln("Failed to create user info table:", err)
            DB.Close()
            return
        }

        DB.CreateContentInfoTable()
        if err != nil {
            log.Errorln("Failed to create content info table:", err)
            DB.Close()
            return
        }
    }
    defer DB.Close()

    err = DB.InsertUserInfo(serverConfig.Auths)
    if err != nil {
        log.Errorln("Failed to add user info to database:", err)
        return
    }

    // Create a new router
    router := NewRouter()
    // Run the router
    go router.run()

    server := http.Server{
        Addr:    serverConfig.Address,
        Handler: nil,
    }

    // Handle requests
    http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
        // Serve the websocket connection
        ServeWs(router, w, r)
    })

    http.HandleFunc("/get", getClipboardHandler)

    quit := make(chan os.Signal, 1)

    go func() {
        err = server.ListenAndServeTLS(serverConfig.Certificate.CertFile, serverConfig.Certificate.KeyFile)
        if err != http.ErrServerClosed {
            log.Errorln("Start service failed:", err)
            quit <- syscall.SIGTERM
        }
    }()

    signal.Notify(quit, os.Interrupt)
    <-quit
    log.Infoln("waiting for shutdown finishing...")
    ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
    defer cancel()
    if err := server.Shutdown(ctx); err != nil {
        log.Fatalf("shutdown server err: %v.", err)
    }
    log.Infoln("shutdown finished.")
}
