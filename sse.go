// Frodo implements an Event Source that supports multiple channels.
//
// Examples
//
// Basic usage of Frodo.
//
//    	es := frodo.NewEventSource()
//
//    	defer es.Shutdown()
//
//    	http.Handle("/{channel:[a-z0-9-_/]+}", es)
//
//		es.SendMessage("channel", "Hello world!")
package sse

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Log interface {
	Info(message string, args ...interface{})
}

type defaultLog struct {
}

func (d *defaultLog) Info(message string, args ...interface{}) {}

type Client struct {
	channel string
	ip      string
	send    chan string
}

type eventMessage struct {
	channel string
	message string
}

type Channel struct {
	mutex            *sync.Mutex
	lastMessage      string
	clientsConnected map[*Client]bool
	expiration       int64
}

type EventSource struct {
	allowCors      bool
	channels       map[string]*Channel
	addClient      chan *Client
	removeClient   chan *Client
	sendMessage    chan *eventMessage
	shutdown       chan bool
	closeChannel   chan string
	log            Log
	UseLastMessage bool
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

func (ch *Channel) GetLastMessage() string {
	return ch.lastMessage
}

func (ch *Channel) SetLastMessage(message string) {
	ch.lastMessage = message
}

func (ch *Channel) UpdateExpiration() {
	ch.mutex.Lock()
	ch.expiration = time.Now().UnixNano()
	ch.mutex.Unlock()
}

func (ch *Channel) Expired() bool {
	if ch.expiration == 0 {
		return false
	}
	ch.mutex.Lock()
	defer ch.mutex.Unlock()
	return (time.Now().UnixNano() - ch.expiration) > (24 * int64(time.Hour))
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

// NewEventSource creates a new Event Source with config.
func NewEventSourceConfig(allowCors, lastMessage bool, log Log) *EventSource {
	es := &EventSource{
		allowCors,
		make(map[string]*Channel),
		make(chan *Client),
		make(chan *Client),
		make(chan *eventMessage),
		make(chan bool),
		make(chan string),
		log,
		lastMessage,
	}

	go es.dispatch()

	if es.UseLastMessage {
		go es.ClearChannels()
	}

	return es
}

// NewEventSource creates a new Event Source.
func NewEventSource() *EventSource {
	return NewEventSourceConfig(true, false, &defaultLog{})
}

func (es *EventSource) ClearChannels() {
	ticker := time.NewTicker(5 * time.Minute)
	for {
		select {
		case <-ticker.C:
			es.DeleteExpired()
		}
	}
}

func (es *EventSource) DeleteExpired() {
	for key, ch := range es.channels {
		if ch.Expired() {
			es.internalCloseChannel(key)
		}
	}
}

func (es *EventSource) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	flusher, ok := response.(http.Flusher)

	if !ok {
		http.Error(response, "Streaming unsupported.", http.StatusInternalServerError)
		return
	}

	h := response.Header()

	if es.allowCors {
		h.Set("Access-Control-Allow-Origin", "*")
		h.Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		h.Set("Access-Control-Allow-Headers", "Keep-Alive,X-Requested-With,Cache-Control,Content-Type,Last-Event-ID")
	}

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

		done := make(chan bool)

		go func() {
			<-notify
			es.removeClient <- c
			done <- true
		}()

		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg := <-c.send:
				{
					fmt.Fprintf(response, "data: %s\n\n", msg)
					flusher.Flush()
				}
			case <-ticker.C:
				{
					fmt.Fprintf(response, ": keep-alive\n\n")
					flusher.Flush()
				}
			case <-done:
				return
			}
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

	for _, channel := range es.channels {
		i += len(channel.clientsConnected)
	}

	return i
}

// ConnectionCountPerChannel returns the connection count in selected channel.
func (es *EventSource) ConnectionCountPerChannel(channel string) int {
	if ch, ok := es.channels[channel]; ok {
		return len(ch.clientsConnected)
	}

	return 0
}

// Shutdown performs a graceful shutdown of Event Source.
func (es *EventSource) Shutdown() {
	es.shutdown <- true
}

func (es *EventSource) internalCloseChannel(channel string) {
	if ch, exists := es.channels[channel]; exists {
		delete(es.channels, channel)

		// Kick all clients of this channel.
		for c, _ := range ch.clientsConnected {
			ch.clientsConnected[c] = false
			delete(ch.clientsConnected, c)
			close(c.send)
		}

	}
}

// dispatch holds all Event Source channels logic.
func (es *EventSource) dispatch() {
	for {
		select {

		// New client connected.
		case c := <-es.addClient:
			ch, exists := es.channels[c.channel]

			if !exists {
				ch = &Channel{
					new(sync.Mutex),
					"",
					make(map[*Client]bool),
					0,
				}
				es.channels[c.channel] = ch
				es.log.Info("New channel '%s' created.", c.channel)
			}

			ch.UpdateExpiration()

			ch.clientsConnected[c] = true
			es.log.Info("Client '%s' connected to channel '%s'.", c.ip, c.channel)
			lastMessage := ch.GetLastMessage()

			if es.UseLastMessage && (len(lastMessage) > 0) {
				c.send <- ch.lastMessage
			}

		// Client disconnected.
		case c := <-es.removeClient:
			if ch, exists := es.channels[c.channel]; exists {
				ch.UpdateExpiration()

				if _, ok := ch.clientsConnected[c]; ok {
					delete(ch.clientsConnected, c)
					close(c.send)
				}

				es.log.Info("Client '%s' disconnected from channel '%s'.", c.ip, c.channel)
				if len(ch.clientsConnected) == 0 {
					es.log.Info("Channel '%s' has no clients. Closing channel.", c.channel)
					es.internalCloseChannel(c.channel)
				}
			}

		// Broadcast message to all clients in channel.
		case msg := <-es.sendMessage:
			es.log.Info("Broadcasting to '%s'", msg.channel)
			if ch, ok := es.channels[msg.channel]; ok {
				for c, open := range ch.clientsConnected {
					if open {
						c.send <- msg.message
					}
				}

				if es.UseLastMessage {
					ch.UpdateExpiration()

					ch.SetLastMessage(msg.message)
				}
				es.log.Info("Message sent to %d clients on channel '%s'.", len(ch.clientsConnected), msg.channel)
			} else {
				es.log.Info("Channel '%s' doesn't exists. Message not sent.", msg.channel)
			}

		// Close channel and all clients in it.
		case channel := <-es.closeChannel:
			es.internalCloseChannel(channel)

		// Event Source shutdown.
		case <-es.shutdown:
			es.CloseChannels()
			close(es.addClient)
			close(es.removeClient)
			close(es.sendMessage)
			close(es.shutdown)
			es.log.Info("Event Source server stoped.")
			return
		}
	}
}
