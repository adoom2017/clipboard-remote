package main

import (
    "flag"
    "fmt"
    "html"
    "net/http"
    "os"
    "path"

    util "clipboard-remote/common"

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

    DB *util.DBInfo
)

func init() {
    // Set the report callers to true
    log.SetReportCaller(true)
    // Set the formatter to include the function name and line number
    log.SetFormatter(&util.Formatter{
        HideKeys:    true,
        CallerFirst: true,
        NoColors:    true,
    })

    // Set the output to the standard output
    log.SetOutput(os.Stdout)

    // Set the log level
    log.SetLevel(log.InfoLevel)
}

func fooHandler(w http.ResponseWriter, r *http.Request) {
    fmt.Fprintf(w, "hello,%q", html.EscapeString(r.URL.Path))
}

func main() {
    // Parse command line arguments
    flag.Parse()

    configPath := path.Join(*configDir, *configFile)
    serverConfig, err := util.ServerConfigRead(configPath)
    if err != nil {
        log.Errorf("Failed to load server config file(%s), err: %v.", configPath, err)
        return
    }

    // Init sqlite database
    dbFilePath := path.Join(*configDir, "server.sqlite3")
    if util.Exists(dbFilePath) {
        DB = util.InitDB(dbFilePath)
    } else {
        DB = util.InitDB(dbFilePath)
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

    // Handle requests
    http.HandleFunc("/websocket", func(w http.ResponseWriter, r *http.Request) {
        // Serve the websocket connection
        ServeWs(router, w, r)
    })

    http.HandleFunc("/hello", fooHandler)

    // Listen and serve HTTPS
    err = http.ListenAndServeTLS(serverConfig.Address, serverConfig.Certificate.CertFile, serverConfig.Certificate.KeyFile, nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
