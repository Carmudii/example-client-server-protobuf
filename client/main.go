package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	pb "client/pokemon"

	"github.com/gorilla/websocket"
	"google.golang.org/protobuf/proto"
)

func main() {
	// Prompt the user for the server URL
	reader := bufio.NewReader(os.Stdin)

	// Create a new WebSocket dialer
	dialer := websocket.DefaultDialer

	// Dial the WebSocket server
	conn, _, err := dialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		log.Println("WebSocket dial error:", err)
		return
	}

	// Read messages from the server
	go readMessages(conn)

	// Prompt the user for input and send it to the server
	fmt.Println(`
	--[ Welcome to the Pokemon WebSocket client ]--
	Enter "list" to list all pokemons.
	Enter "get id <id>" to get a pokemon by id.
	Enter "get name <name>" to get a pokemon by name.
	Enter "get region <region>" to get a pokemon by region.
	Enter "exit" to exit.`)

	for {
		// Sleep for a second to prevent spamming
		time.Sleep(10 * time.Millisecond)

		fmt.Print("Enter command: ")
		command, _ := reader.ReadString('\n')
		command = strings.TrimSuffix(command, "\n")

		var queryCommand *pb.PokemonQuery

		switch {
		case command == "list":
			fmt.Printf("Listing all pokemons...\n")
			queryCommand = &pb.PokemonQuery{}

		case strings.HasPrefix(command, "get id "):
			arg := strings.TrimPrefix(command, "get id ")

			if _, err := strconv.Atoi(arg); err == nil {
				fmt.Printf("Getting pokemon by id %s...\n", arg)
				queryCommand = &pb.PokemonQuery{Id: arg}

			} else {
				fmt.Println("Invalid command argument. Usage: get id <id>")

			}
		case strings.HasPrefix(command, "get name "):
			arg := strings.TrimPrefix(command, "get name ")
			fmt.Printf("Getting pokemon by name %s...\n", arg)
			queryCommand = &pb.PokemonQuery{Name: arg}

		case strings.HasPrefix(command, "get region "):
			arg := strings.TrimPrefix(command, "get region ")
			fmt.Printf("Getting pokemon by region %s...\n", arg)
			queryCommand = &pb.PokemonQuery{Region: arg}

		case command == "exit":
			fmt.Println("Exiting...")
			return
		default:
			fmt.Println("Invalid command")
		}

		query, error := proto.Marshal(queryCommand)
		if error != nil {
			log.Println("protobuf encode error:", error)
			continue
		}

		sendMessage(conn, query)
	}
}

// Read messages from the WebSocket connection
func readMessages(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("WebSocket read error:", err)
			return
		}

		// Decode the message as a PokemonList
		newWebSocketMessage := &pb.WebSocketMessage{}
		err = proto.Unmarshal(message, newWebSocketMessage)
		if err != nil {
			log.Println("protobuf decode error:", err)
			continue
		}

		switch newWebSocketMessage.GetPaylod().(type) {
		case *pb.WebSocketMessage_PokemonList:
			fmt.Println("Received Pokemons:")
			for _, p := range newWebSocketMessage.GetPokemonList().Pokemon {
				fmt.Printf("Name: %s, id: %s, type: %s\n", p.Name, p.Id, p.Type)
			}

		case *pb.WebSocketMessage_ErrorMessage:
			fmt.Println("[SERVER]:", newWebSocketMessage.GetErrorMessage().ErrorMessage)

		default:
			fmt.Println("undefined message type")
		}
	}
}

// Send a message to the WebSocket connection
func sendMessage(conn *websocket.Conn, msg []byte) {
	err := conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("WebSocket write error:", err)
	}
}
