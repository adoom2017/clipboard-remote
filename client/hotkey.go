package main

import (
  "clipboard-remote/clipboard"
  "clipboard-remote/utils"
  "context"
  "crypto/tls"
  "encoding/json"
  "fmt"
  "io"
  "net/http"

  log "github.com/sirupsen/logrus"
  "golang.design/x/hotkey"
)

type Hotkey struct {
  client *Client
}

func registerHotkey(key hotkey.Key, mods ...hotkey.Modifier) (*hotkey.Hotkey, error) {
  modifier := []hotkey.Modifier{}
  modifier = append(modifier, mods...)
  hk := hotkey.New(modifier, key)

  err := hk.Register()
  if err != nil {
    return nil, err
  }

  return hk, nil
}

func (h *Hotkey) listenHotkey(ctx context.Context) (err error) {
  upKey, err := registerHotkey(hotkey.KeyC, hotkey.ModAlt)
  if err != nil {
    log.Errorln("Failed to register upload key:", err)
    return nil
  }

  downKey, err := registerHotkey(hotkey.KeyV, hotkey.ModAlt)
  if err != nil {
    log.Errorln("Failed to register download key:", err)
    return nil
  }

  for {
    select {
    case <-upKey.Keydown():
      h.uploadHotkeyHandler()
    case <-downKey.Keydown():
      h.downloadHotkeyHandler()
    case <-ctx.Done():
      log.Infoln("Exit hotkey listen routine.")
      return
    }
  }
}

func (h *Hotkey) uploadHotkeyHandler() {
  // read clipboard content
  clipBuff := clipboard.Read()
  if clipBuff == nil {
    log.Errorln("Failed to read clipboard.")
    return
  }

  // send clipboard content to server
  h.client.writeCh <- &utils.WebsocketMessage{
    Action: utils.ActionClipboardChanged,
    UserID: h.client.ID,
    Data:   clipBuff,
  }
}

func (h *Hotkey) downloadHotkeyHandler() {
  transport := &http.Transport{
    TLSClientConfig: &tls.Config{InsecureSkipVerify: h.client.config.InsecureSkipVerify},
  }
  client := &http.Client{Transport: transport}

  url := fmt.Sprintf("https://%s:%d/clipboard/get", h.client.config.Host, h.client.config.Port)
  req, err := http.NewRequest("GET", url, nil)
  if err != nil {
    log.Errorln("Failed to handle new request:", err)
    return
  }

  req.SetBasicAuth(h.client.config.Auth.User, h.client.config.Auth.Password)
  resp, err := client.Do(req)
  if err != nil {
    log.Errorln("Failed to request remote clipboard info:", err)
    return
  }
  defer resp.Body.Close()

  bodyText, err := io.ReadAll(resp.Body)
  if err != nil {
    log.Errorln("Failed to read clipboard info:", err)
    return
  }

  var respInfo utils.RespInfo
  err = json.Unmarshal(bodyText, &respInfo)
  if err != nil {
    log.Errorln("Failed to decode response info:", err)
    return
  }

  buff := utils.ClipBoardBuff{
    Type: utils.CLIP_TEXT,
    Buff: utils.StringToBytes(respInfo.Data.Content),
  }
  buffClip, err := utils.EncodeToBytes(buff)
  if err != nil {
    log.Errorln("Failed to decode buff:", err)
    return
  }

  clipboard.Write(buffClip)
}
