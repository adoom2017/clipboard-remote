package main

import (
  "clipboard-remote/utils"
  "encoding/base64"
  "encoding/json"
  "html/template"
  "io"
  "net/http"

  log "github.com/sirupsen/logrus"
)

type ClipHandler struct {
  router *Router
  //respWriter http.ResponseWriter
  //req        *http.Request
}

func NewClipHandler(r *Router) *ClipHandler {
  return &ClipHandler{router: r}
}

func GetSessionUser(r *http.Request) string {
  session, _ := SessionStore.Get(r, "session-id")
  s, ok := session.Values["user"]
  if !ok {
    return ""
  }

  return s.(string)
}

func SaveSessionUser(w http.ResponseWriter, r *http.Request, username string) {
  session, _ := SessionStore.Get(r, "session-id")

  session.Values["user"] = username
  session.Options.HttpOnly = true
  session.Options.Secure = true

  SessionStore.Save(r, w, session)
}

type RestfulRespInfo struct {
  Response utils.RespInfo
  Writer   http.ResponseWriter // http response writer
}

func (rest *RestfulRespInfo) send() {
  b, _ := json.Marshal(rest.Response)

  rest.Writer.Header().Set("Content-Type", "application/json")
  rest.Writer.WriteHeader(rest.Response.Code)
  rest.Writer.Write(b)
}

// UserBasicAuthMDW http basic authentication middleware func
func UserBasicAuthMDW(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    user := GetSessionUser(r)

    // restful API reponse sender
    rest := RestfulRespInfo{
      Writer: w,
      Response: utils.RespInfo{
        Code:    http.StatusOK,
        Message: "Succeed.",
      },
    }

    // never login
    if user == "" {
      // authentication process
      user, passwd, ok := r.BasicAuth()
      if !ok {
        log.Errorln("No Basic Authentication Info.")

        w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

        rest.Response.Code = http.StatusUnauthorized
        rest.Response.Message = "Authentication Failed."

        rest.send()
        return
      }

      pass := DB.GetPassword(user)

      // if no password find means user not exist
      if pass == "" || pass != passwd {
        log.Errorln("Failed to authentication user:", user)

        w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)

        rest.Response.Code = http.StatusUnauthorized
        rest.Response.Message = "Authentication Failed."

        rest.send()
        return
      }

      // use session for application
      SaveSessionUser(w, r, user)
    }

    next.ServeHTTP(w, r)
  })
}

// RestGetClipHandler Get clipboard content handler for restful API
func (clip *ClipHandler) RestGetClipHandlerFunc(w http.ResponseWriter, r *http.Request) {
  user := GetSessionUser(r)

  // restful API reponse sender
  rest := RestfulRespInfo{
    Writer: w,
    Response: utils.RespInfo{
      Code:    http.StatusOK,
      Message: "Get clipboard succeed.",
    },
  }

  defer rest.send()

  buff, err := base64.StdEncoding.DecodeString(DB.GetClipContentByName(user))
  if err != nil {
    log.Errorln("Failed to get clipboard content for user:", user, err)

    rest.Response.Code = http.StatusInternalServerError
    rest.Response.Message = "Get Clipboard Content Failed."
    return
  }

  content, err := utils.DecodeToStruct(buff)
  if err != nil {
    log.Errorln("Failed to get clipboard content for user:", user, err)

    rest.Response.Code = http.StatusInternalServerError
    rest.Response.Message = "Get Clipboard Content Failed."
    return
  }

  if content.Type != utils.CLIP_TEXT {
    rest.Response.Code = http.StatusInternalServerError
    rest.Response.Message = "Get Clipboard Content Failed."
    return
  }

  rest.Response.Data = &utils.DataInfo{
    Type:    "text",
    Content: utils.BytesToString(content.Buff),
  }
}

// RestGetClipHandler Set clipboard content handler for restful API
func (clip *ClipHandler) RestSetClipHandlerFunc(w http.ResponseWriter, r *http.Request) {
  user := GetSessionUser(r)

  // restful API reponse sender
  rest := RestfulRespInfo{
    Writer: w,
    Response: utils.RespInfo{
      Code:    http.StatusOK,
      Message: "Set clipboard succeed.",
    },
  }

  defer rest.send()

  // Get content
  body, err := io.ReadAll(r.Body)
  if err != nil {
    rest.Response.Code = http.StatusInternalServerError
    rest.Response.Message = err.Error()
    return
  }

  var dataInfo utils.DataInfo
  if err = json.Unmarshal(body, &dataInfo); err != nil {
    rest.Response.Code = http.StatusInternalServerError
    rest.Response.Message = err.Error()
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
}

// LoginHandlerFunc handler for login action
func (clip *ClipHandler) DoLoginHandlerFunc(w http.ResponseWriter, r *http.Request) {
  user := GetSessionUser(r)

  // never login
  if user == "" {
    // authentication process
    r.ParseForm()

    user := r.FormValue("username")
    passwd := r.FormValue("password")
    //remember := r.FormValue("remember-check")

    pass := DB.GetPassword(user)

    if pass == "" || pass != passwd {
      log.Errorf("Failed to auth user(%s).", user)

      t, err := template.ParseFiles("../static/login.html")
      if err != nil {
        log.Println(err)
      }

      t.Execute(w, "用户名或者密码错误，请重新登录！")
      return
    }

    SaveSessionUser(w, r, user)
  }

  http.Redirect(w, r, "/content", http.StatusFound)
}

// LoginHtmlHandlerFunc handler for login html page
func (clip *ClipHandler) LoginHtmlHandlerFunc(w http.ResponseWriter, r *http.Request) {
  u := GetSessionUser(r)
  if u != "" {
    http.Redirect(w, r, "/content", http.StatusFound)
    return
  }

  t, err := template.ParseFiles("../static/login.html")
  if err != nil {
    log.Errorln("Failed to parse template file: ", err)
    return
  }

  t.Execute(w, nil)
}

// ContentHtmlHandlerFunc handler for content html page
func (clip *ClipHandler) ContentHtmlHandlerFunc(w http.ResponseWriter, r *http.Request) {
  user := GetSessionUser(r)

  // never login
  if user == "" {
    http.Redirect(w, r, "/", http.StatusFound)
    return
  }

  t, err := template.ParseFiles("../static/content.html")
  if err != nil {
    log.Println(err)
  }

  var clipInfos []DisplayInfo
  contents := DB.GetClipContents()
  for _, content := range contents {
    buff, _ := base64.StdEncoding.DecodeString(content.Content)
    clipInfo, _ := utils.DecodeToStruct(buff)

    clipInfos = append(clipInfos, DisplayInfo{
      ClientID:  content.ClientID,
      Timestamp: content.Timestamp,
      UserName:  content.Username,
      Content:   utils.BytesToString(clipInfo.Buff),
    })
  }

  t.Execute(w, clipInfos)
}

// WsHandlerFunc handler for websocket action
func (clip *ClipHandler) WsHandlerFunc(w http.ResponseWriter, r *http.Request) {
  ServeWs(clip.router, w, r)
}
