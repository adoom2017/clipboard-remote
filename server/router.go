package main

import (
  "clipboard-remote/utils"
  "container/list"
)

// Router maintains the set of active clients and broadcasts messages to the
// clients.
type Router struct {
  // Registered clients.
  clients map[string]*list.List

  // Message from the client, need broadcast to others.
  broadcast chan *Message

  // Unregister request from client.
  unregister chan *Client

  // Register request from client
  register chan *Client
}

// Message info
type Message struct {
  // Message sender's client ID
  id string

  // Message sender's client username
  username string

  // message content
  content []byte
}

// NewRouter return router instance
func NewRouter() *Router {
  return &Router{
    broadcast:  make(chan *Message),
    unregister: make(chan *Client),
    register:   make(chan *Client),
    clients:    make(map[string]*list.List),
  }
}

// run is the main loop of the router
func (r *Router) run() {
  for {
    select {
    // register client
    case client := <-r.register:
      if tmpList, ok := r.clients[client.username]; ok {
        tmpList.PushBack(client)
      } else {
        tmpList = list.New()
        tmpList.PushBack(client)
        r.clients[client.username] = tmpList
      }
    // unregister client
    case client := <-r.unregister:
      if tmpList, ok := r.clients[client.username]; ok {
        for i := tmpList.Front(); i != nil; i = i.Next() {
          if tmp := i.Value.(*Client); tmp.id == client.id {
            tmpList.Remove(i)

            // close client send buffer
            close(tmp.send)
            break
          }
        }
      }
    // broadcast client message
    case message := <-r.broadcast:
      if tmpList, ok := r.clients[message.username]; ok {
        for i := tmpList.Front(); i != nil; i = i.Next() {
          if tmp := i.Value.(*Client); message.id == tmp.id {
            continue
          } else {
            // add content to other client send buffer
            wsm := &utils.WebsocketMessage{
              Action: utils.ActionClipboardChanged,
              UserID: message.id,
              Data:   message.content,
            }

            tmp.send <- wsm.Encode()
          }
        }
      }
    }
  }
}
