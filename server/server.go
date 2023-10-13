package main

import (
  "clipboard-remote/utils"
  "context"
  "flag"
  "net/http"
  "os"
  "os/signal"
  "path"
  "path/filepath"
  "syscall"
  "time"

  "github.com/gorilla/websocket"
  log "github.com/sirupsen/logrus"
)

var (
  configDir  = flag.String("d", "", "server config directory")
  configFile = flag.String("f", "", "server config file")

  upgrader = websocket.Upgrader{
    ReadBufferSize:    4096,
    WriteBufferSize:   4096,
    EnableCompression: true,
  }

  // database handler
  DB *utils.DBInfo

  // server config
  GlobalConfig *utils.ServerConfig
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

func main() {
  // Parse command line arguments
  flag.Parse()

  tmpHomeDir := *configDir
  if tmpHomeDir != "" {
    if !filepath.IsAbs(tmpHomeDir) {
      currentDir, _ := os.Getwd()
      tmpHomeDir = filepath.Join(currentDir, tmpHomeDir)
    }
  }

  tmpConfigFile := *configFile
  if tmpConfigFile != "" {
    if !filepath.IsAbs(tmpConfigFile) {
      currentDir, _ := os.Getwd()
      tmpConfigFile = filepath.Join(currentDir, tmpConfigFile)
    }
  } else {
    tmpConfigFile = filepath.Join(tmpHomeDir, "server.yaml")
  }

  tmpConfig, err := utils.ServerConfigRead(tmpConfigFile)
  if err != nil {
    log.Errorf("Failed to load server config file(%s), err: %v.", tmpConfigFile, err)
    return
  }

  GlobalConfig = tmpConfig

  // Set the log level from config file
  switch GlobalConfig.Log.LogLevel {
  case "debug":
    log.SetLevel(log.DebugLevel)
  case "info":
    log.SetLevel(log.InfoLevel)
  case "warn":
    log.SetLevel(log.WarnLevel)
  case "error":
    log.SetLevel(log.ErrorLevel)
  case "fatal":
    log.SetLevel(log.FatalLevel)
  }

  // Set the log file
  if GlobalConfig.Log.LogPath != "" {
    logfile, err := os.OpenFile(GlobalConfig.Log.LogPath, os.O_RDONLY|os.O_CREATE, 0666)
    if err != nil {
      log.Errorf("Failed to open log file(%s), err: %v.", GlobalConfig.Log.LogPath, err)
      // print the log info to stdout
    } else {
      log.SetOutput(logfile)
    }
  }

  // Init sqlite database
  dbFilePath := path.Join(tmpHomeDir, "server.sqlite3")
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

  err = DB.InsertUserInfo(GlobalConfig.Auths)
  if err != nil {
    log.Errorln("Failed to add user info to database:", err)
    return
  }

  // Create a new router
  router := NewRouter()
  // Run the router
  go router.run()

  server := http.Server{
    Addr:    GlobalConfig.Address,
    Handler: nil,
  }

  // Handle requests
  http.HandleFunc(GlobalConfig.WebsocketPath, func(w http.ResponseWriter, r *http.Request) {
    // Serve the websocket connection
    ServeWs(router, w, r)
  })

  http.HandleFunc("/clipboard/get", getClipboardHandler)
  http.HandleFunc("/clipboard/set", func(w http.ResponseWriter, r *http.Request) {
    setClipboardHandler(router, w, r)
  })

  quit := make(chan os.Signal, 1)

  go func() {
    err = server.ListenAndServeTLS(GlobalConfig.Certificate.CertFile, GlobalConfig.Certificate.KeyFile)
    if err != http.ErrServerClosed {
      log.Errorln("Start service failed:", err)
      quit <- syscall.SIGTERM
    }
  }()

  log.Infoln("Server start succeed.")

  signal.Notify(quit, os.Interrupt)
  <-quit
  log.Infoln("Waiting for shutdown finishing...")
  ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
  defer cancel()
  if err := server.Shutdown(ctx); err != nil {
    log.Fatalf("Shutdown server err: %v.", err)
  }
  log.Infoln("Server shutdown succeed.")
}
