package common

import (
    "os"
    "testing"

    "gopkg.in/yaml.v3"
)

func TestConfigRead(t *testing.T) {
    tmp := Config{
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

    config, err := ConfigRead("tmp.yaml")
    if err != nil {
        t.Error("Failed read config.")
        return
    }

    if config.Auth.User != tmp.Auth.User || config.Log.LogLevel != "info" || config.WebsocketPath != "/" {
        t.Fatal("Check failed.")
    }

    os.Remove("tmp.yaml")
}
