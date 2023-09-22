package common

import (
    "os"
    "testing"
)

func TestUserDB(t *testing.T) {
    db := InitDB("test-user.sqlite3")
    if db == nil {
        t.Fatal("Failed to init sqlite.")
    }
    defer os.Remove("test-user.sqlite3")
    defer db.Close()

    err := db.CreateUserInfoTable()
    if err != nil {
        t.Fatal("Failed to create user info table:", err)
    }

    auths := []AuthConfig{
        {User: "test1", Password: "password1"},
        {User: "test1", Password: "password2"},
        {User: "test3", Password: "password3"},
    }

    err = db.InsertUserInfo(auths)
    if err != nil {
        t.Fatal("Failed to insert user info table:", err)
    }

    alls := db.GetUsers()
    if len(alls) != 2 {
        t.Fatal("Num Error:", len(alls))
    }

    auth := db.GetUserByName("test1")
    if auth.Password != "password2" || auth.User != "test1" {
        t.Fatal("User get failed.")
    }
}

func TestContentDB(t *testing.T) {
    db := InitDB("test-content.sqlite3")
    if db == nil {
        t.Fatal("Failed to init sqlite.")
    }
    defer os.Remove("test-content.sqlite3")
    defer db.Close()

    err := db.CreateContentInfoTable()
    if err != nil {
        t.Fatal("Failed to create content info table:", err)
    }

    contents := []ClipContentInfo{
        {ClientID: "11", Username: "u1", Content: "content11"},
        {ClientID: "21", Username: "u2", Content: "content21"},
        {ClientID: "22", Username: "u2", Content: "content22"},
        {ClientID: "31", Username: "u3", Content: "content31"},
        {ClientID: "32", Username: "u3", Content: "content32"},
        {ClientID: "32", Username: "u3", Content: "content333"},
    }

    for _, content := range contents {
        err = db.InsertClipContent(&content)
        if err != nil {
            t.Fatal("Failed to insert user info table:", err)
        }
    }

    temp := db.GetClipContentByID("32")
    if temp != "content333" {
        t.Fatal("Get content by id failed:", temp)
    }

    temp = db.GetClipContentByName("u2")
    if temp != "content22" {
        t.Fatal("Get content by name failed:", temp)
    }

    tempList := db.GetClipContents()
    if len(tempList) != 5 {
        t.Fatal("Num Error:", len(tempList))
    }
}
