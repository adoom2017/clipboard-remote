package main

import (
  "context"
  "flag"
  "os"
  "os/signal"
  "path/filepath"
  "time"

  "clipboard-remote/clipboard"
  "clipboard-remote/utils"

  log "github.com/sirupsen/logrus"
)

var (
  configDir  = flag.String("d", "", "client config directory")
  configFile = flag.String("f", "", "client config file path")
)

func init() {
  // Set the log output to stdout
  log.SetReportCaller(true)
  log.SetFormatter(&utils.Formatter{
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
    tmpConfigFile = filepath.Join(tmpHomeDir, "client.yaml")
  }

  clientConfig, err := utils.ClientConfigRead(tmpConfigFile)
  if err != nil {
    log.Errorf("Failed to load client config file(%s), err: %v.", tmpConfigFile, err)
    return
  }

  // Set the log level from config file
  switch clientConfig.Log.LogLevel {
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
  if clientConfig.Log.LogPath != "" {
    logfile, err := os.OpenFile(clientConfig.Log.LogPath, os.O_RDONLY|os.O_CREATE, 0666)
    if err != nil {
      log.Errorf("Failed to open log file(%s), err: %v.", clientConfig.Log.LogPath, err)
      // print the log info to stdout
    } else {
      log.SetOutput(logfile)
    }
  }

  // add interrupt sigal
  interrupt := make(chan os.Signal, 1)
  signal.Notify(interrupt, os.Interrupt)

  // watch context, used for watch break
  ctx, cancel := context.WithCancel(context.Background())

  // handle io local to server
  client := NewClient(clientConfig)

  if clientConfig.Mode == "auto" {
    go client.handleIO(ctx, clipboard.Watch(ctx))
  } else {
    hk := &Hotkey{
      client:         client,
      hotkeyUpload:   clientConfig.HotKey.UploadKey,
      hotkeyDownload: clientConfig.HotKey.DownloadKey,
    }

    go client.handleIO(ctx, nil)
    go hk.listenHotkey(ctx)
  }

  <-interrupt
  cancel()
  log.Infoln("Waiting client to exist...")

  client.close()
  time.Sleep(1 * time.Second)
}
