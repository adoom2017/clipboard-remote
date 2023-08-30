package main

// Router maintains the set of active clients and broadcasts messages to the
// clients.
type Router struct {
    // Registered clients.
    clients map[*Server]bool

    // Inbound messages from the clients.
    broadcast chan *Message

    // Register requests from the clients.
    register chan *Server

    // Unregister requests from clients.
    unregister chan *Server
}

// Message info
type Message struct {
    // Message Send Client ID
    id string

    // message content
    content []byte
}

// NewRouter return router instance
func NewRouter() *Router {
    return &Router{
        broadcast:  make(chan *Message),
        register:   make(chan *Server),
        unregister: make(chan *Server),
        clients:    make(map[*Server]bool),
    }
}

// run is the main loop of the router
func (r *Router) run() {
    for {
        select {
        case client := <-r.register:
            r.clients[client] = true
        case client := <-r.unregister:
            if _, ok := r.clients[client]; ok {
                delete(r.clients, client)
                close(client.send)
            }
        case message := <-r.broadcast:
            for client := range r.clients {
                if message.id == client.id {
                    continue
                }

                select {
                case client.send <- message.content:
                default:
                    close(client.send)
                    delete(r.clients, client)
                }
            }
        }
    }
}