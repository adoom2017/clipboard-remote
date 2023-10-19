package utils

import (
  "os"
  "testing"

  "gopkg.in/yaml.v3"
)

func TestClientConfigRead(t *testing.T) {
  tmp := ClientConfig{
    Host: "127.0.0.1",
    Port: 443,
    Auth: AuthConfig{
      User:     "test-user",
      Password: "test-password",
    },
    InsecureSkipVerify: false,
    Mode:               "auto",
  }

  buff, err := yaml.Marshal(tmp)
  if err != nil {
    t.Error("Marshal Failed:", err)
    return
  }

  err = os.WriteFile("tmp.yaml", buff, 0666)
  if err != nil {
    t.Error("Write tmp file failed:", err)
    return
  }

  config, err := ClientConfigRead("tmp.yaml")
  if err != nil {
    t.Error("Failed read config.")
    return
  }

  if config.Auth.User != tmp.Auth.User || config.Log.LogLevel != "info" || config.WebsocketPath != "/" {
    t.Fatal("Check failed.")
  }

  os.Remove("tmp.yaml")
}

func TestServerConfigRead(t *testing.T) {
  tmp := ServerConfig{
    Auths: []AuthConfig{
      {User: "user1", Password: "password1"},
      {User: "user2", Password: "password2"},
      {User: "user3", Password: "password3"},
    },
    MaxMsgSize:    1024,
    Address:       "0.0.0.0:443",
    WebsocketPath: "/ws",
    Certificate: CertConfig{
      CertFile: "ssl.crt",
      KeyFile:  "ssl.key",
    },
    Log: LogConfig{
      LogPath:  "./sss.log",
      LogLevel: "warn",
    },
  }

  buff, err := yaml.Marshal(tmp)
  if err != nil {
    t.Error("Marshal Failed:", err)
    return
  }

  err = os.WriteFile("tmp.yaml", buff, 0666)
  if err != nil {
    t.Error("Write tmp file failed:", err)
    return
  }

  config, err := ServerConfigRead("tmp.yaml")
  if err != nil {
    t.Error("Failed read config.")
    return
  }

  if len(config.Auths) != 3 {
    t.FailNow()
  }

  if config.Auths[2].User != "user3" || config.Auths[2].Password != "password3" {
    t.FailNow()
  }

  if config.Log.LogLevel != "warn" {
    t.FailNow()
  }

  os.Remove("tmp.yaml")
}
