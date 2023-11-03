package main

import (
  "clipboard-remote/utils"
  "encoding/base64"
  "encoding/json"
  "io"
  "net/http"
  "strings"
  "text/template"

  "github.com/google/uuid"
  log "github.com/sirupsen/logrus"
)

type ClipHandler struct {
  router     *Router
  respWriter http.ResponseWriter
  req        *http.Request
}

func NewClipHandler(r *Router) *ClipHandler {
  return &ClipHandler{router: r}
}

func SessionID() string {
  uuid_s := uuid.New().String()

  return strings.ReplaceAll(uuid_s, "-", "")
}

func (clip *ClipHandler) authBasicHTTP() string {
  user, passwd, ok := clip.req.BasicAuth()
  if !ok {
    return ""
  }

  pass := DB.GetPassword(user)

  // if no password find means user not exist
  if pass == "" || pass != passwd {
    log.Errorf("Failed to auth user(%s).", user)
    return ""
  }

  return user
}

func (clip *ClipHandler) sendResponse(buff []byte, statusCode int) {
  clip.respWriter.Header().Set("Content-Type", "application/json")
  clip.respWriter.WriteHeader(http.StatusBadRequest)
  clip.respWriter.Write(buff)
}

func (clip *ClipHandler) authFailedResponse() {
  buff, _ := json.Marshal(utils.RespInfo{
    Code:    http.StatusUnauthorized,
    Message: "Authentication failed.",
  })

  clip.sendResponse(buff, http.StatusUnauthorized)
}

func (clip *ClipHandler) normalFailedResponse(msg string) {
  buff, _ := json.Marshal(utils.RespInfo{
    Code:    http.StatusBadRequest,
    Message: msg,
  })

  clip.sendResponse(buff, http.StatusBadRequest)
}

// RestGetClipHandler Get clipboard content handler for restful API
func (clip *ClipHandler) RestGetClipHandlerFunc(w http.ResponseWriter, r *http.Request) {
  clip.req = r
  clip.respWriter = w

  user := clip.authBasicHTTP()
  if user == "" {
    log.Errorln("Authentication failed.")
    clip.authFailedResponse()
    return
  }

  buff, err := base64.StdEncoding.DecodeString(DB.GetClipContentByName(user))
  if err != nil {
    log.Errorln("Failed to get clipboard content for user:", user, err)
    clip.normalFailedResponse("Get clipboard info failed.")
    return
  }

  content, err := utils.DecodeToStruct(buff)
  if err != nil {
    log.Errorln("Failed to get clipboard content for user:", user, err)
    clip.normalFailedResponse("Get clipboard info failed.")
    return
  }

  if content.Type != utils.CLIP_TEXT {
    clip.normalFailedResponse("Unsupported content type.")
    return
  }

  respInfo := utils.RespInfo{
    Code:    http.StatusOK,
    Message: "Get clipboard succeed.",
    Data: &utils.DataInfo{
      Type:    "text",
      Content: utils.BytesToString(content.Buff),
    },
  }

  result, _ := json.Marshal(respInfo)

  clip.sendResponse(result, http.StatusOK)
}

// RestGetClipHandler Set clipboard content handler for restful API
func (clip *ClipHandler) RestSetClipHandlerFunc(w http.ResponseWriter, r *http.Request) {
  user := clip.authBasicHTTP()
  if user == "" {
    log.Errorln("Authentication failed.")
    clip.authFailedResponse()
    return
  }

  // Get content
  body, err := io.ReadAll(r.Body)
  if err != nil {
    clip.normalFailedResponse(err.Error())
    return
  }

  var dataInfo utils.DataInfo
  if err = json.Unmarshal(body, &dataInfo); err != nil {
    clip.normalFailedResponse(err.Error())
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
  clip.router.broadcast <- &Message{
    id:       dataInfo.ClientID,
    username: user,
    content:  clipBuff,
  }

  respInfo := utils.RespInfo{
    Code:    http.StatusOK,
    Message: "Set clipboard succeed.",
  }

  buff, _ := json.Marshal(respInfo)

  clip.sendResponse(buff, http.StatusOK)
}

// LoginHandlerFunc handler for login action
func (clip *ClipHandler) DoLoginHandlerFunc(w http.ResponseWriter, r *http.Request) {
  r.ParseForm()

  user := r.FormValue("username")
  passwd := r.FormValue("password")
  //remember := r.FormValue("remember-check")

  pass := DB.GetPassword(user)

  if pass == "" || pass != passwd {
    log.Errorf("Failed to auth user(%s).", user)

    t, err := template.ParseFiles("../static/login_fail.html")
    if err != nil {
      log.Println(err)
    }

    t.Execute(w, "用户名或者密码错误！！！")

    return
  }

  http.Redirect(w, r, "/content", http.StatusFound)
}

// LoginHtmlHandlerFunc handler for login html page
func (clip *ClipHandler) LoginHtmlHandlerFunc(w http.ResponseWriter, r *http.Request) {
  t, err := template.ParseFiles("../static/login.html")
  if err != nil {
    log.Println(err)
  }

  t.Execute(w, nil)
}

// ContentHtmlHandlerFunc handler for content html page
func (clip *ClipHandler) ContentHtmlHandlerFunc(w http.ResponseWriter, r *http.Request) {
  t, err := template.ParseFiles("../static/content.html")
  if err != nil {
    log.Println(err)
  }

  var clipInfos []DisplayInfo
  contents := DB.GetClipContents()
  for _, content := range contents {
    buff, _ := base64.StdEncoding.DecodeString(content.Content)
    clipInfo, _ := utils.DecodeToStruct(buff)

    clipInfos = append(clipInfos, DisplayInfo{UserName: content.Username, Content: utils.BytesToString(clipInfo.Buff)})
  }

  t.Execute(w, clipInfos)
}

// WsHandlerFunc handler for websocket action
func (clip *ClipHandler) WsHandlerFunc(w http.ResponseWriter, r *http.Request) {
  ServeWs(clip.router, w, r)
}
