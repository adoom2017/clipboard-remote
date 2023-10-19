package utils

import (
  "encoding/json"
  "errors"
)

// Errors
var (
  ErrBadAction             = errors.New("bad action data")
  ErrAuthFailed            = errors.New("auth failed")
  ErrUnAuthenticatedClient = errors.New("client is not authenticated")
)

// All actions from daemons
const (
  ActionNone              WebsocketAction = "none"
  ActionHandshakeRegister WebsocketAction = "register"
  ActionHandshakeReady                    = "ready"
  ActionClipboardChanged                  = "cbchanged"
  ActionClipboardGet                      = "cbget"
  ActionClipboardPut                      = "cbput"
  ActionTerminate                         = "terminate"
)

// WebsocketAction is an action between midgard daemon and midgard server
type WebsocketAction string

// WebsocketMessage represents a message for websocket.
type WebsocketMessage struct {
  Action WebsocketAction `json:"action"`
  UserID string          `json:"user_id"`
  Data   []byte          `json:"data"`
}

// Encode encodes a websocket message
func (m *WebsocketMessage) Encode() []byte {
  b, _ := json.Marshal(m)
  return b
}

// Decode decodes given data to m.
func (m *WebsocketMessage) Decode(data []byte) error {
  return json.Unmarshal(data, m)
}
