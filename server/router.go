package main

// Router maintains the set of active clients and broadcasts messages to the
// clients.
type Router struct {
    // Registered clients.
    clients map[*Client]bool

    // Inbound messages from the clients.
    broadcast chan *Message

    // Register requests from the clients.
    register chan *Client

    // Unregister requests from clients.
    unregister chan *Client
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
        register:   make(chan *Client),
        unregister: make(chan *Client),
        clients:    make(map[*Client]bool),
    }
}

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
