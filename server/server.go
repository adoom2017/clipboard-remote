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
    log.SetReportCaller(true)
    log.SetFormatter(&util.Formatter{
        HideKeys:    true,
        CallerFirst: true,
        NoColors:    true,
    })

    log.SetOutput(os.Stdout)

    //log.SetLevel(log.WarnLevel)
    log.SetLevel(log.InfoLevel)
}

func main() {
    flag.Parse()
    router := NewRouter()
    go router.run()

    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        ServeWs(router, w, r)
    })

    err := http.ListenAndServeTLS(*addr, "../certificate/ssl.crt", "../certificate/ssl.key", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}
