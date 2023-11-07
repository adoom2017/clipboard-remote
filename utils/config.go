package utils

import (
  "os"

  log "github.com/sirupsen/logrus"

  "gopkg.in/yaml.v3"
)

// ClientConfig clipboard client config
type ClientConfig struct {
  Host               string       `yaml:"server"`
  Port               int          `yaml:"port"`
  WebsocketPath      string       `yaml:"websocket-path"`
  Log                LogConfig    `yaml:"log"`
  Auth               AuthConfig   `yaml:"auth"`
  InsecureSkipVerify bool         `yaml:"skip-cert-verify"`
  HotKey             HotKeyConfig `yaml:"hotkey"`
  Mode               string       `yaml:"mode"`
}

// ServerConfig clipboard server config
type ServerConfig struct {
  Auths         []AuthConfig  `yaml:"auths"`
  MaxMsgSize    int           `yaml:"max-msg-size"`
  Address       string        `yaml:"addr"`
  WebsocketPath string        `yaml:"websocket-path"`
  Certificate   CertConfig    `yaml:"certificate"`
  Log           LogConfig     `yaml:"log"`
  Session       SessionConfig `yaml:"session"`
}

type SessionConfig struct {
  Key    string `yaml:"key"`
  MaxAge int    `yaml:"max-age"`
}

// CertConfig config the certificate files
type CertConfig struct {
  CertFile string `yaml:"cert-file"`
  KeyFile  string `yaml:"key-file"`
}

// LogConfig log config, default is stdOut, level is info
type LogConfig struct {
  LogPath  string `yaml:"path"`
  LogLevel string `yaml:"log-level"`
}

// AuthConfig auth config
type AuthConfig struct {
  User     string `yaml:"user"`
  Password string `yaml:"password"`
}

type HotKeyConfig struct {
  UploadKey   string `yaml:"upload"`
  DownloadKey string `yaml:"download"`
}

// ClientConfigRead read the client config, and set default value
func ClientConfigRead(configFile string) (*ClientConfig, error) {
  // Read the config file
  data, err := os.ReadFile(configFile)
  if err != nil {
    log.Errorln("Falied to read config file:", configFile)
    return nil, err
  }

  // Parse config file
  var config ClientConfig
  err = yaml.Unmarshal(data, &config)
  if err != nil {
    log.Errorln("Failed to parse config file.")
    return nil, err
  }

  // Set the default value
  if config.WebsocketPath == "" {
    config.WebsocketPath = "/"
  }

  if config.Log.LogLevel == "" {
    config.Log.LogLevel = "info"
  }

  if config.HotKey.UploadKey == "" {
    config.HotKey.UploadKey = "Alt+C"
  }

  if config.HotKey.DownloadKey == "" {
    config.HotKey.DownloadKey = "Alt+V"
  }

  return &config, nil
}

// ServerConfigRead read the server config, and set default value
func ServerConfigRead(configFile string) (*ServerConfig, error) {
  // Read the config file
  data, err := os.ReadFile(configFile)
  if err != nil {
    log.Errorln("Falied to read config file:", configFile)
    return nil, err
  }

  // Parse config file
  var config ServerConfig
  err = yaml.Unmarshal(data, &config)
  if err != nil {
    log.Errorln("Failed to parse config file.")
    return nil, err
  }

  // Set the default value
  if config.WebsocketPath == "" {
    config.WebsocketPath = "/"
  }

  if config.Log.LogLevel == "" {
    config.Log.LogLevel = "info"
  }

  if config.MaxMsgSize == 0 {
    config.MaxMsgSize = 10 * 1024 * 1024
  }

  if config.Session.Key == "" {
    config.Session.Key = "90po()PO12wq!@WQ"
  }

  if config.Session.MaxAge == 0 {
    config.Session.MaxAge = 3600
  }

  return &config, nil
}
