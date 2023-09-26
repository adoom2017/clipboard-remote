package main

// Router maintains the set of active clients and broadcasts messages to the
// clients.
type Router struct {
    // Registered clients.
    clients map[*Server]string

    // Inbound messages from the clients.
    broadcast chan *Message

    // Unregister requests from clients.
    unregister chan *Server
}

// Message info
type Message struct {
    // Message Send Client ID
    id string

    // Message from user
    username string

    // message content
    content []byte
}

// NewRouter return router instance
func NewRouter() *Router {
    return &Router{
        broadcast:  make(chan *Message),
        unregister: make(chan *Server),
        clients:    make(map[*Server]string),
    }
}

// run is the main loop of the router
func (r *Router) run() {
    for {
        select {
        case client := <-r.unregister:
            if _, ok := r.clients[client]; ok {
                delete(r.clients, client)
                close(client.send)
            }
        case message := <-r.broadcast:
            for client := range r.clients {
                if message.id == client.id || message.username != client.username {
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
