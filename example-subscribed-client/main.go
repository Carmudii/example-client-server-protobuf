package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	pb "subscribed-client/protobuf"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func main() {
	// Create a new reader to read user input
	reader := bufio.NewReader(os.Stdin)

	// Create a new WebSocket dialer
	dialer := websocket.DefaultDialer

	// Copy this to the terminal to test the client
	// subs positive
	// subs negative
	// unsubs positive
	// unsubs negative

	// Dial the WebSocket server
	conn, _, err := dialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Println("WebSocket dial error:", err)
		return
	}

	go readMessage(conn)

	for {
		// Sleep for a second to prevent spamming
		time.Sleep(10 * time.Millisecond)

		fmt.Print("Enter command: ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSuffix(command, "\n")

		switch {
		case strings.HasPrefix(command, "unsubs "):
			// TODO: Implement unsubs command

		case strings.HasPrefix(command, "subs "):
			// TODO: Implement subs command

		case command == "exit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid command")
		}

		// Create a new WebSocket message
		queryCommand := pb.SubscribeRequest{
			Channel: command,
		}

		// Marshal the message
		msg, err := proto.Marshal(&queryCommand)
		if err != nil {
			log.Println("Proto marshal error:", err)
			return
		}

		sendMessage(conn, msg)
	}
}

func readMessage(conn *websocket.Conn) {
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			return
		}

		switch messageType {
		case websocket.TextMessage:
			fmt.Println(string(message))

		case websocket.BinaryMessage:
			var webSocketMessage pb.WebSocketMessage
			if err := proto.Unmarshal(message, &webSocketMessage); err != nil {
				log.Println("Proto unmarshal error:", err)
				return
			}

			fmt.Printf("[SERVER]: %s\n", webSocketMessage.GetSubscribeResponse().Message)
		}
	}
}

// Send a message to the WebSocket connection
func sendMessage(conn *websocket.Conn, msg []byte) {
	err := conn.WriteMessage(websocket.BinaryMessage, msg)
	if err != nil {
		log.Println("WebSocket write error:", err)
	}
}
