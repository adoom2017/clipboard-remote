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
  "strings"

  log "github.com/sirupsen/logrus"
  "golang.design/x/hotkey"
)

type Hotkey struct {
  client         *Client
  hotkeyUpload   string
  hotKeyDownload string
}

func switchKeys(keyString string) (hotkey.Key, []hotkey.Modifier) {
  hotkeys := strings.Split(keyString, "+")

  var tempModKeys []hotkey.Modifier
  for i := 0; i < len(hotkeys)-1; i++ {
    switch hotkeys[i] {
    case "Control":
      fallthrough
    case "CONTROL":
      tempModKeys = append(tempModKeys, hotkey.ModCtrl)
    case "Alt":
      fallthrough
    case "ALT":
      tempModKeys = append(tempModKeys, hotkey.ModAlt)
    case "Shift":
      fallthrough
    case "SHIFT":
      tempModKeys = append(tempModKeys, hotkey.ModShift)
    }
  }

  var tempKey hotkey.Key
  switch hotkeys[len(hotkeys)-1] {
  case "A":
    tempKey = hotkey.KeyA
  case "B":
    tempKey = hotkey.KeyB
  case "C":
    tempKey = hotkey.KeyC
  case "D":
    tempKey = hotkey.KeyD
  case "E":
    tempKey = hotkey.KeyE
  case "F":
    tempKey = hotkey.KeyF
  case "G":
    tempKey = hotkey.KeyG
  case "H":
    tempKey = hotkey.KeyH
  case "I":
    tempKey = hotkey.KeyI
  case "J":
    tempKey = hotkey.KeyJ
  case "K":
    tempKey = hotkey.KeyK
  case "L":
    tempKey = hotkey.KeyL
  case "M":
    tempKey = hotkey.KeyM
  case "N":
    tempKey = hotkey.KeyN
  case "O":
    tempKey = hotkey.KeyO
  case "P":
    tempKey = hotkey.KeyP
  case "Q":
    tempKey = hotkey.KeyQ
  case "R":
    tempKey = hotkey.KeyR
  case "S":
    tempKey = hotkey.KeyS
  case "T":
    tempKey = hotkey.KeyT
  case "U":
    tempKey = hotkey.KeyU
  case "V":
    tempKey = hotkey.KeyV
  case "W":
    tempKey = hotkey.KeyW
  case "X":
    tempKey = hotkey.KeyX
  case "Y":
    tempKey = hotkey.KeyY
  case "Z":
    tempKey = hotkey.KeyZ
  case "Space":
    tempKey = hotkey.KeySpace
  }

  return tempKey, tempModKeys
}

func registerHotkey(key hotkey.Key, mods []hotkey.Modifier) (*hotkey.Hotkey, error) {
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

  tmpKey, tmpMods := switchKeys(h.hotkeyUpload)
  upKey, err := registerHotkey(tmpKey, tmpMods)
  if err != nil {
    log.Errorln("Failed to register upload key:", err)
    return nil
  }

  tmpKey, tmpMods = switchKeys(h.hotKeyDownload)
  downKey, err := registerHotkey(tmpKey, tmpMods)
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
