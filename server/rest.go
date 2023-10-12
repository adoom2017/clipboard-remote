package main

import (
  "clipboard-remote/utils"
  "encoding/base64"
  "encoding/json"
  "io"
  "net/http"
  "strings"

  log "github.com/sirupsen/logrus"
)

type RespInfo struct {
  Code    int       `json:"code"`
  Message string    `json:"message"`
  Data    *DataInfo `json:"data,omitempty"`
}

type DataInfo struct {
  ClientID string `json:"client_id,omitempty"`
  Type     string `json:"type,omitempty"`
  Content  string `json:"content,omitempty"`
}

func authBasicHTTP(token string) string {
  if len(token) <= 6 {
    log.Errorln("Invalid token:", token)
    return ""
  }

  authInfo, err := base64.StdEncoding.DecodeString(token[len("Basic "):])
  if err != nil {
    log.Errorln("Token decode failed:", err)
    return ""
  }

  tokens := strings.Split(utils.BytesToString(authInfo), ":")
  if len(tokens) != 2 {
    log.Errorln("Invalid token:", authInfo)
    return ""
  }
  user := tokens[0]
  pass := DB.GetPassword(user)

  if pass != tokens[1] {
    log.Errorf("Failed to auth user(%s).", user)
    return ""
  }

  return user
}

func authFailedResponse(w http.ResponseWriter) {
  buff, _ := json.Marshal(RespInfo{
    Code:    http.StatusUnauthorized,
    Message: "Authentication failed.",
  })

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusUnauthorized)
  w.Write(buff)
}

func normalFailedResponse(w http.ResponseWriter, msg string) {
  buff, _ := json.Marshal(RespInfo{
    Code:    http.StatusBadRequest,
    Message: msg,
  })

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusBadRequest)
  w.Write(buff)
}

func getSucceedResponse(w http.ResponseWriter, respBuff []byte) {
  respInfo := RespInfo{
    Code:    http.StatusOK,
    Message: "Get clipboard succeed.",
    Data: &DataInfo{
      Type:    "text",
      Content: utils.BytesToString(respBuff),
    },
  }

  buff, _ := json.Marshal(respInfo)

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  w.Write(buff)
}

func setSucceedResponse(w http.ResponseWriter) {
  respInfo := RespInfo{
    Code:    http.StatusOK,
    Message: "Set clipboard succeed.",
  }

  buff, _ := json.Marshal(respInfo)

  w.Header().Set("Content-Type", "application/json")
  w.WriteHeader(http.StatusOK)
  w.Write(buff)
}

func getClipboardHandler(w http.ResponseWriter, r *http.Request) {
  user := authBasicHTTP(r.Header.Get("Authorization"))
  if user == "" {
    log.Errorln("Authentication failed.")
    authFailedResponse(w)
    return
  }

  buff, err := base64.StdEncoding.DecodeString(DB.GetClipContentByName(user))
  if err != nil {
    log.Errorln("Failed to get clipboard content for user:", user, err)
    normalFailedResponse(w, "Get clipboard info failed.")
    return
  }

  content, err := utils.DecodeToStruct(buff)
  if err != nil {
    log.Errorln("Failed to get clipboard content for user:", user, err)
    normalFailedResponse(w, "Get clipboard info failed.")
    return
  }

  if content.Type != utils.CLIP_TEXT {
    normalFailedResponse(w, "Unsupported content type.")
    return
  }

  getSucceedResponse(w, content.Buff)
}

func setClipboardHandler(router *Router, w http.ResponseWriter, r *http.Request) {
  user := authBasicHTTP(r.Header.Get("Authorization"))
  if user == "" {
    log.Errorln("Authentication failed.")
    authFailedResponse(w)
    return
  }

  // Get content
  body, err := io.ReadAll(r.Body)
  if err != nil {
    normalFailedResponse(w, err.Error())
    return
  }

  var dataInfo DataInfo
  if err = json.Unmarshal(body, &dataInfo); err != nil {
    normalFailedResponse(w, err.Error())
    return
  }

  clipBuff, _ := utils.EncodeToBytes(utils.ClipBoardBuff{
    Type: utils.CLIP_TEXT,
    Buff: utils.StringToBytes(dataInfo.Content),
  })

  // insert clipboard data into database
  err = DB.InsertClipContent(&utils.ClipContentInfo{
    ClientID: dataInfo.ClientID,
    Username: user,
    Content:  base64.StdEncoding.EncodeToString(clipBuff),
  })

  if err != nil {
    log.Errorf("Failed to insert clipcontent to database, id: %s, user: %s.", dataInfo.ClientID, user)
    // Ignore errors; therefore, no return
  }

  // broadcast clipboard content to user's all client
  router.broadcast <- &Message{
    id:       dataInfo.ClientID,
    username: user,
    content:  clipBuff,
  }

  setSucceedResponse(w)
}
