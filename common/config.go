package common

import (
    "os"

    log "github.com/sirupsen/logrus"

    "github.com/google/uuid"
    "gopkg.in/yaml.v3"
)

// Config clipboard client config
type Config struct {
    Host               string     `yaml:"server"`
    Port               int        `yaml:"port"`
    WebsocketPath      string     `yaml:"websocket-path"`
    Log                LogConfig  `yaml:"log"`
    Auth               AuthConfig `yaml:"auth"`
    Uuid               string     `yaml:"uuid"`
    InsecureSkipVerify bool       `yaml:"skip-cert-verify"`
    Mode               string     `yaml:"mode"`
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

// ConfigRead read the client config, and set default value
func ConfigRead(configFile string) (*Config, error) {
    // Read the config file
    data, err := os.ReadFile(configFile)
    if err != nil {
        log.Errorln("Falied to read config file:", configFile)
        return nil, err
    }

    // Parse config file
    var config Config
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

    if config.Uuid == "" {
        config.Uuid = uuid.New().String()
    }

    return &config, nil
}
