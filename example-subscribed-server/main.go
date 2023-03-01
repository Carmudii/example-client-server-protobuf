package main

/*
 * It is a simple websocket server that will publish a random number every second
 * with a specific channel and will send it to all clients where subscribed to that channel
 * build with protocol buffers
 */

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"

	pb "handle-subscribed/protobuf"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"
)

type WebSocketServer struct {
	clients    map[*websocket.Conn]bool
	broadcast  chan map[string][]byte
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	channels   map[*websocket.Conn]map[string]bool
}

// This a method that will handle subscription requests coming from the client
func (server *WebSocketServer) subscribe(client *websocket.Conn, channel string) {
	if server.clients[client] {
		if server.channels[client] == nil {
			server.channels[client] = make(map[string]bool)
		}

		server.channels[client][channel] = true
	}
}

// This a method that will handle unsubscribe requests coming from the client
func (server *WebSocketServer) unsubscribe(client *websocket.Conn, channel string) {
	if server.clients[client] {
		if server.channels[client] == nil {
			server.channels[client] = make(map[string]bool)
		}

		server.channels[client][channel] = false
	}
}

// This a method that allows us to broadcast a message to all clients with specific channel
// subscribed to it
func (server *WebSocketServer) broadcastToSubscribers(channel string, message *[]byte) {
	for client, channelMap := range server.channels {
		// Check if the user is have subscribed to the channel
		if channelMap[channel] {
			err := client.WriteMessage(websocket.BinaryMessage, *message)
			if err != nil {
				fmt.Println("Error writing message:", err)
			}
		}
	}
}

func (server *WebSocketServer) sendTextMessage(conn *websocket.Conn, message *[]byte) {
	err := conn.WriteMessage(websocket.TextMessage, *message)
	if err != nil {
		fmt.Println("Error writing message:", err)
	}
}

// This a method that will handle websocket requests coming from the client
func (server *WebSocketServer) handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {

	fmt.Println("New client connected:", r.RemoteAddr)

	// Upgrade initial GET request to a websocket
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Upgrade the connection to a websocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading connection to websocket:", err)
		return
	}

	// Register our new client
	server.register <- conn

	var message = []byte(`---[ Welcome to subscribed-client ]---
	Command list:
	subs <channel>
	unsubs <channel>
	channel list: positive, negative`)

	// Send initial message to the client
	server.sendTextMessage(conn, &message)

	// Make sure we close the connection when the function returns
	defer func() {
		server.unregister <- conn
		conn.Close()
	}()

	// Listen indefinitely for new messages coming
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Unmarshal the request
		var request pb.SubscribeRequest
		err = proto.Unmarshal(msg, &request)
		if err != nil {
			fmt.Println("Error unmarshaling request:", err)
			continue
		}

		fmt.Println("Received message:", request.Channel)

		// Subscribe or unsubscribe the client to the channel
		// based on the request
		if request.Channel == "unsubs positive" {
			server.unsubscribe(conn, "positive")

		} else if request.Channel == "unsubs negative" {
			server.unsubscribe(conn, "negative")

		} else if request.Channel == "subs positive" {
			server.subscribe(conn, "positive")

		} else if request.Channel == "subs negative" {
			server.subscribe(conn, "negative")
		}
	}
}

// This a goroutine that will run in the background and will
// publish a random number every second and send it to all clients where subscribed
func (server *WebSocketServer) publisher() {
	for {

		// Generate a random price
		positive := 10.00 + rand.Float64()*(100.00-10.00)
		negative := -10.00 + rand.Float64()*(-100.00+10.00)

		// Create a response
		var positiveResponse = pb.WebSocketMessage{
			Paylod: &pb.WebSocketMessage_SubscribeResponse{
				SubscribeResponse: &pb.SubscribeResponse{
					Message: fmt.Sprint(positive),
				},
			},
		}

		var negativeResponse = pb.WebSocketMessage{
			Paylod: &pb.WebSocketMessage_SubscribeResponse{
				SubscribeResponse: &pb.SubscribeResponse{
					Message: fmt.Sprint(negative),
				},
			},
		}

		// Check if the connection is still active before sending the price
		// ok := server.clients[conn]
		// if !ok {
		// 	break
		// }

		fmt.Println("Total client connected:", len(server.clients))

		// Marshal the response
		postiveMessage, err := proto.Marshal(&positiveResponse)
		if err != nil {
			fmt.Println("Error marshaling response:", err)
			break
		}

		negativeMessage, err := proto.Marshal(&negativeResponse)
		if err != nil {
			fmt.Println("Error marshaling response:", err)
			break
		}

		// Send send message to specific channel
		// where channel is key of a map and value is a message
		var messageMap = make(map[string][]byte)
		messageMap["positive"] = postiveMessage
		messageMap["negative"] = negativeMessage

		// Send the message to the client
		server.broadcast <- messageMap

		// Sleep for a second before sending the next update
		time.Sleep(time.Second)
	}
}

// This a goroutine that will run in the background
func (server *WebSocketServer) run() {
	for {
		select {
		case conn := <-server.register:
			// Register the new client
			server.clients[conn] = true

		case conn := <-server.unregister:
			// Check if the connection is still active before unregistering it
			if ok := server.clients[conn]; ok {
				delete(server.clients, conn)
				delete(server.channels, conn)
			}

		case message := <-server.broadcast:
			// Send the message to all clients that are subscribed to the channel
			for channel, byte := range message {
				server.broadcastToSubscribers(channel, &byte)
			}
		}
	}
}

func main() {
	// Create a new server
	server := WebSocketServer{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan map[string][]byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		channels:   make(map[*websocket.Conn]map[string]bool),
	}

	// Setup route
	http.HandleFunc("/ws", server.handleWebSocketConnection)

	// Start the server
	go server.run()

	// Start the publisher
	go server.publisher()

	log.Println("Starting server...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
