package main

import (
  "clipboard-remote/utils"
  "testing"
)

func TestHotkey(t *testing.T) {
  clientConfig, err := utils.ClientConfigRead("../config/client.yaml")
  if err != nil {
    t.Error("Failed to read config:", err)
    t.Failed()
    return
  }

  client := NewClient(clientConfig)

  hotkey := &Hotkey{client: client}

  hotkey.downloadHotkeyHandler()
}
