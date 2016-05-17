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

type Client struct {
    channel string
    ip string
    send chan string
}

type eventMessage struct {
    channel string
    message string
}

type EventSource struct {
    channels map[string]map[*Client]bool
    addClient chan *Client
    removeClient chan *Client
    sendMessage chan *eventMessage
    shutdown chan bool
    closeChannel chan string

    ClientConnect chan *Client
    ClientDisconnect chan *Client
    ChannelCreate chan string
    ChannelClose chan string
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
func (c *Client) SendMessage(message string) {
    c.send <- message
}

// Channel returns the channel where this client is subscribe to.
func (c *Client) Channel() string {
    return c.channel
}

// IP return the client IP.
func (c *Client) IP() string {
    return c.ip
}

// NewEventSource creates a new Event Source.
func NewEventSource() *EventSource {
    es := &EventSource{
        make(map[string]map[*Client]bool),
        make(chan *Client),
        make(chan *Client),
        make(chan *eventMessage),
        make(chan bool),
        make(chan string),

        make(chan *Client),
        make(chan *Client),
        make(chan string),
        make(chan string),
    }

    go es.dispatch()

    return es
}

func (es *EventSource) ServeHTTP(response http.ResponseWriter, request *http.Request) {
    flusher, ok := response.(http.Flusher)

    if !ok {
        http.Error(response, "Streaming unsupported.", http.StatusInternalServerError)
        return
    }

    h := response.Header()

    if request.Method == "GET" {
        h.Set("Content-Type", "text/event-stream")
        h.Set("Cache-Control", "no-cache")
        h.Set("Connection", "keep-alive")

        c := &Client{
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

        for msg := range c.send {
            fmt.Fprintf(response, "data: %s\n\n", msg)
            flusher.Flush()
        }
    }
}

// SendMessage broadcast a message to all clients in a channel.
func (es *EventSource) SendMessage(channel, message string) {
    if es.ChannelExists(channel) {
        msg := &eventMessage{
            channel,
            message,
        }

        es.sendMessage <- msg
    }
}

// Channels returns all opened channels name, or an empty array if none open.
func (es *EventSource) Channels() []string {
    channels := make([]string, 0)

    for channel, _ := range es.channels {
        channels = append(channels, channel)
    }

    return channels
}

// Channels returns true if a channel exists, false otherwise.
func (es *EventSource) ChannelExists(channel string) bool {
    _, ok := es.channels[channel]
    return ok
}

// CloseChannel closes an especific channel and all clients in it.
func (es *EventSource) CloseChannel(channel string) {
    es.closeChannel <- channel
}

// CloseChannels closes all channels and clients.
func (es *EventSource) CloseChannels() {
    for channel, _ := range es.channels {
        es.closeChannel <- channel
    }
}

// ConnectionCount returns the connection count in Event Source.
func (es *EventSource) ConnectionCount() int {
    i := 0

    for _, clients := range es.channels {
        i += len(clients)
    }

    return i
}

// ConnectionCountPerChannel returns the connection count in selected channel.
func (es *EventSource) ConnectionCountPerChannel(channel string) int {
    if ch, ok := es.channels[channel]; ok {
        return len(ch)
    }

    return 0
}

// Shutdown performs a graceful shutdown of Event Source.
func (es *EventSource) Shutdown() {
    es.shutdown <- true
}

// dispatch holds all Event Source channels logic.
func (es *EventSource) dispatch() {
    for {
        select {

        // New client connected.
        case c := <- es.addClient:
            ch, exists := es.channels[c.channel]

            if !exists {
                ch = make(map[*Client]bool)
                es.channels[c.channel] = ch
                log.Printf("New channel '%s' created.\n", c.channel)
                es.ChannelCreate <- c.channel
            }

            ch[c] = true
            log.Printf("Client '%s' connected to channel '%s'.\n", c.ip, c.channel)
            es.ClientConnect <- c

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
                    es.ChannelClose <- c.channel
                }
            }

            close(c.send)
            es.ClientDisconnect <- c

        // Broadcast message to all clients in channel.
        case msg := <- es.sendMessage:
            if ch, ok := es.channels[msg.channel]; ok {
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

                // Kick all clients of this channel.
                for c, _ := range ch {
                    ch[c] = false
                    delete(ch, c)
                    close(c.send)
                }

                log.Printf("Channel '%s' closed.\n", channel)
            } else {
                log.Printf("Requested to close channel '%s', but it was already closed.\n", channel)
            }

        // Event Source shutdown.
        case <- es.shutdown:
            es.CloseChannels()
            close(es.addClient)
            close(es.removeClient)
            close(es.sendMessage)
            close(es.shutdown)
            close(es.ClientConnect)
            close(es.ClientDisconnect)
            close(es.ChannelCreate)
            close(es.ChannelClose)
            log.Println("Event Source server stoped.")
            return
        }
    }
}
