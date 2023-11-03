package main

import (
  "fmt"
  "net/http"

  "github.com/gorilla/securecookie"
  "github.com/gorilla/sessions"
)

var (
  store = sessions.NewFilesystemStore("./", securecookie.GenerateRandomKey(32), securecookie.GenerateRandomKey(32))
)

func Set(w http.ResponseWriter, r *http.Request) {
  session, _ := store.Get(r, "user")
  session.Values["name"] = "dj"
  session.Values["age"] = 18
  err := sessions.Save(r, w)
  if err != nil {
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  fmt.Fprintln(w, "Hello World")
}

func Read(w http.ResponseWriter, r *http.Request) {
  session, _ := store.Get(r, "user")
  fmt.Fprintf(w, "name:%s age:%d\n", session.Values["name"], session.Values["age"])
}
