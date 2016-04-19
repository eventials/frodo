// Package sse implements an Event Source that supports multiple channels.
//
// Examples
//
// Basic usage of sse package.
//
//    es := sse.NewEventSource(func (c sse.Client) {
//        c.SendMessage(fmt.Sprintf("Greetings %s!", c.IP()))
//    }, nil)
//
//    defer es.Shutdown()
//
//    http.Handle("/events/", es)
//
package sse

import (
    "fmt"
    "log"
    "net"
    "net/http"
    "strings"
)

type Client interface {
    SendMessage(message string) // Send message to client.
    Channel() string            // The channel where this client is subscribe to.
    IP() string                 // Client IP.
}

type EventSource interface {
    Channels() []string                // Returns all opened channels name, or an empty array if none open.
    GetLastMessage(channel string) (string, bool)
    SetLastMessage(channel, message string)
    ChannelExists(channel string) bool // Returns true if a channel exists, false otherwise.
    CloseChannel(channel string)       // Closes an especific channel and all clients in it.
    CloseChannels()                    // Closes all channels and clients.
    ConnectionCountPerChannel(channel string) int     // Returns the connection count in selected channel.
    ServeHTTP(w http.ResponseWriter, r *http.Request) //
    SendMessage(channel, message string) // Broadcast message to all clients in a channel.
    ConnectionCount() int                // Returns the connection count in Event Source.
    Shutdown()                           // Performs a graceful shutdown of Event Source.
}

type Settings struct {
    AllowCors bool
    OnClientConnect func (EventSource, Client)
    OnClientDisconnect func (EventSource, Client)
    OnChannelCreate func (EventSource, string)
    OnChannelClose func (EventSource, string)
}

type client struct {
    channel string
    ip string
    send chan string
}

type eventMessage struct {
    channel string
    message string
}

type eventSource struct {
    channels map[string]map[*client]bool
    lastMessage map[string]string
    addClient chan *client
    removeClient chan *client
    sendMessage chan *eventMessage
    shutdown chan bool
    closeChannel chan string
    settings Settings
}

func getIP(request *http.Request) string {
    ip := request.Header.Get("X-Forwarded-For")

    if len(ip) > 0 {
        return strings.Split(ip, ",")[0]
    }


    if ip, _, err := net.SplitHostPort(request.RemoteAddr); err == nil {
        return ip
    }

    return ""
}

// SendMessage sends a message to client.
func (c *client) SendMessage(message string) {
    c.send <- message
}

// Channel returns the channel where this client is subscribe to.
func (c *client) Channel() string {
    return c.channel
}

// IP return the client IP.
func (c *client) IP() string {
    return c.ip
}

// NewEventSource creates a new Event Source.
func NewEventSource(settings Settings) EventSource {
    es := &eventSource{
        make(map[string]map[*client]bool),
        make(map[string]string),
        make(chan *client),
        make(chan *client),
        make(chan *eventMessage),
        make(chan bool),
        make(chan string),
        settings,
    }

    go es.dispatch()

    return es
}

func (es *eventSource) ServeHTTP(response http.ResponseWriter, request *http.Request) {
    flusher, ok := response.(http.Flusher)

    if !ok {
        http.Error(response, "Streaming unsupported.", http.StatusInternalServerError)
        return
    }

    c := &client{
        request.URL.Path, // Channel name is the full path.
        getIP(request),
        make(chan string),
    }

    es.addClient <- c

    notify := response.(http.CloseNotifier).CloseNotify()

    go func() {
        <-notify
        es.removeClient <- c
    }()

    h := response.Header()
    h.Set("Content-Type", "text/event-stream")
    h.Set("Cache-Control", "no-cache")
    h.Set("Connection", "keep-alive")

    if es.settings.AllowCors {
        h.Set("Access-Control-Allow-Origin", "*")
    }

    for {
        msg := <-c.send
        fmt.Fprintf(response, "data: %s\n\n", msg)
        flusher.Flush()
    }
}

// SendMessage broadcast a message to all clients in a channel.
func (es *eventSource) SendMessage(channel, message string) {
    if es.ChannelExists(channel) {
        msg := &eventMessage{
            channel,
            message,
        }

        es.sendMessage <- msg
    }
}

// Channels returns all opened channels name, or an empty array if none open.
func (es *eventSource) Channels() []string {
    channels := make([]string, 0)

    for channel, _ := range es.channels {
        channels = append(channels, channel)
    }

    return channels
}

func (es *eventSource) GetLastMessage(channel string) (string, bool) {
    msg, ok := es.lastMessage[channel]
    return msg, ok
}

func (es *eventSource) SetLastMessage(channel, message string) {
    es.lastMessage[channel] = message
}

// Channels returns true if a channel exists, false otherwise.
func (es *eventSource) ChannelExists(channel string) bool {
    _, ok := es.channels[channel]
    return ok
}

// CloseChannel closes an especific channel and all clients in it.
func (es *eventSource) CloseChannel(channel string) {
    es.closeChannel <- channel
}

// CloseChannels closes all channels and clients.
func (es *eventSource) CloseChannels() {
    for channel, _ := range es.channels {
        es.closeChannel <- channel
    }
}

// ConnectionCount returns the connection count in Event Source.
func (es *eventSource) ConnectionCount() int {
    i := 0

    for _, clients := range es.channels {
        i += len(clients)
    }

    return i
}

// ConnectionCountPerChannel returns the connection count in selected channel.
func (es *eventSource) ConnectionCountPerChannel(channel string) int {
    if ch, ok := es.channels[channel]; ok {
        return len(ch)
    }

    return 0
}

// Shutdown performs a graceful shutdown of Event Source.
func (es *eventSource) Shutdown() {
    es.shutdown <- true
}

// dispatch holds all Event Source channels logic.
func (es *eventSource) dispatch() {
    for {
        select {

        // New client connected.
        case c := <- es.addClient:
            ch, exists := es.channels[c.channel]

            if !exists {
                ch = make(map[*client]bool)
                es.channels[c.channel] = ch
                log.Printf("New channel '%s' created.\n", c.channel)

                if es.settings.OnChannelCreate != nil {
                    es.settings.OnChannelCreate(es, c.channel)
                }
            }

            ch[c] = true
            log.Printf("Client '%s' connected to channel '%s'.\n", c.ip, c.channel)

            if es.settings.OnClientConnect != nil {
                es.settings.OnClientConnect(es, c)
            }

        // Client disconnected.
        case c := <- es.removeClient:
            if ch, exists := es.channels[c.channel]; exists {
                ch[c] = false
                delete(ch, c)

                log.Printf("Client '%s' disconnected from channel '%s'.\n", c.ip, c.channel)
                log.Printf("Checking if channel '%s' has clients.\n", c.channel)

                if len(ch) == 0 {
                    delete(es.channels, c.channel)
                    log.Printf("Channel '%s' has no clients. Channel closed.\n", c.channel)

                    if es.settings.OnChannelClose != nil {
                        es.settings.OnChannelClose(es, c.channel)
                    }
                }
            }

            close(c.send)

            if es.settings.OnClientDisconnect != nil {
                es.settings.OnClientDisconnect(es, c)
            }

        // Broadcast message to all clients in channel.
        case msg := <- es.sendMessage:
            if ch, ok := es.channels[msg.channel]; ok {
                es.SetLastMessage(msg.channel, msg.message)

                for c, open := range ch {
                    if open {
                        c.send <- msg.message
                    }
                }

                log.Printf("Message sent to %d clients on channel '%s'.\n", len(ch), msg.channel)
            } else {
                log.Printf("Channel '%s' doesn't exists. Message not sent.\n", len(ch), msg.channel)
            }

        // Close channel and all clients in it.
        case channel := <- es.closeChannel:
            if ch, exists := es.channels[channel]; exists {
                delete(es.channels, channel)
                delete(es.lastMessage, channel)

                // Kick all clients of this channel.
                for c, _ := range ch {
                    ch[c] = false
                    delete(ch, c)
                    close(c.send)
                }

                log.Printf("Channel '%s' closed.\n", channel)
            } else {
                log.Printf("Requested to close channel '%s', but it was already closed.", channel)
            }

        // Event Source shutdown.
        case <- es.shutdown:
            es.CloseChannels()
            close(es.addClient)
            close(es.removeClient)
            close(es.sendMessage)
            close(es.shutdown)
            log.Println("Event Source server stoped.")
            return
        }
    }
}
