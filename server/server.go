package main

import (
    "flag"
    "net/http"
    "os"

    util "clipboard-remote/common"

    "github.com/gorilla/websocket"
    log "github.com/sirupsen/logrus"
)

var (
    addr     = flag.String("addr", "0.0.0.0:443", "https service address")
    upgrader = websocket.Upgrader{
        ReadBufferSize:    4096,
        WriteBufferSize:   4096,
        EnableCompression: true,
    }
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

func main() {
    // Parse command line arguments
    flag.Parse()
    // Create a new router
    router := NewRouter()
    // Run the router
    go router.run()

    // Handle requests
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // Serve the websocket connection
        ServeWs(router, w, r)
    })

    // Listen and serve HTTPS
    err := http.ListenAndServeTLS(*addr, "../certificate/ssl.crt", "../certificate/ssl.key", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
