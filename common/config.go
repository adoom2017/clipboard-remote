package common

import (
	"os"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// ClientConfig clipboard client config
type ClientConfig struct {
	Host               string     `yaml:"server"`
	Port               int        `yaml:"port"`
	WebsocketPath      string     `yaml:"websocket-path"`
	Log                LogConfig  `yaml:"log"`
	Auth               AuthConfig `yaml:"auth"`
	Uuid               string     `yaml:"uuid"`
	InsecureSkipVerify bool       `yaml:"skip-cert-verify"`
	Mode               string     `yaml:"mode"`
}

// ServerConfig clipboard server config
type ServerConfig struct {
	Auths         []AuthConfig `yaml:"auths"`
	MaxMsgSize    int          `yaml:"max-msg-size"`
	ListenPort    int          `yaml:"listen"`
	WebsocketPath string       `yaml:"websocket-path"`
	Certificate   CertConfig   `yaml:"certificate"`
	Log           LogConfig    `yaml:"log"`
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

	if config.Uuid == "" {
		config.Uuid = uuid.New().String()
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

	return &config, nil
}
