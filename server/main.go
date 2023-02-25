package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/golang/protobuf/proto"
	"github.com/gorilla/websocket"

	pb "server/pokemon"
)

// Define a global list of Pokemon
var pokemonList = &pb.PokemonList{
	Pokemon: []*pb.Pokemon{
		{Id: "1", Name: "Bulbasaur", Type: "Grass/Poison", Region: "Kanto"},
		{Id: "2", Name: "Ivysaur", Type: "Grass/Poison", Region: "Kanto"},
		{Id: "3", Name: "Venusaur", Type: "Grass/Poison", Region: "Kanto"},
		{Id: "4", Name: "Charmander", Type: "Fire", Region: "Kanto"},
		{Id: "5", Name: "Charmeleon", Type: "Fire", Region: "Kanto"},
		{Id: "6", Name: "Charizard", Type: "Fire/Flying", Region: "Kanto"},
		{Id: "7", Name: "Squirtle", Type: "Water", Region: "Kanto"},
		{Id: "8", Name: "Wartortle", Type: "Water", Region: "Kanto"},
		{Id: "9", Name: "Blastoise", Type: "Water", Region: "Kanto"},
		{Id: "10", Name: "Caterpie", Type: "Bug", Region: "Kanto"},
	},
}

// Define a struct to hold the WebSocket connections
type Connection struct {
	ws   *websocket.Conn
	send chan []byte
}

// Define a method to send a message
func (conn *Connection) handleOutgoingMessage() {
	for message := range conn.send {
		err := conn.ws.WriteMessage(websocket.BinaryMessage, message)
		if err != nil {
			log.Println("Error writing message to WebSocket:", err)
			break
		}
	}
	// Connection is closed
	conn.ws.WriteMessage(websocket.CloseMessage, []byte{})
}

// Define a method to send initial data to the client
func (conn *Connection) sendInitialData() {
	var serverMessage = `--[ Welcome to the Pokemon WebSocket client ]--
	Enter "list" to list all pokemons.
	Enter "get id <id>" to get a pokemon by id.
	Enter "get name <name>" to get a pokemon by name.
	Enter "get region <region>" to get a pokemon by region.
	Enter "exit" to exit.`

	err := conn.ws.WriteMessage(websocket.TextMessage, []byte(serverMessage))
	if err != nil {
		log.Println("Error writing message to WebSocket:", err)
		return
	}
}

// Define a function to convert a PokemonList to a byte slice
func marshalPokemonList(pl *pb.PokemonList) ([]byte, error) {
	var wrappedMessage = &pb.WebSocketMessage{
		Paylod: &pb.WebSocketMessage_PokemonList{
			PokemonList: pl,
		},
	}

	return proto.Marshal(wrappedMessage)
}

// Define a function to convert an error message to a byte slice
func marshalErrorMessage(message string, errorCode int32) ([]byte, error) {
	var wrappedMessage = &pb.WebSocketMessage{
		Paylod: &pb.WebSocketMessage_ErrorMessage{
			ErrorMessage: &pb.ErrorMessage{
				ErrorMessage: message,
				ErrorCode:    errorCode,
			},
		},
	}

	return proto.Marshal(wrappedMessage)
}

// Define a function to handle WebSocket connections
func handleConnection(ws *websocket.Conn, connections map[*Connection]bool) {
	// Create a new connection
	conn := &Connection{
		ws:   ws,
		send: make(chan []byte),
	}
	connections[conn] = true

	// Send initial data to the client
	conn.sendInitialData()

	go conn.handleOutgoingMessage()

	// Listen for incoming messages from client
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			// Remove the connection from the list and close the WebSocket
			delete(connections, conn)
			close(conn.send)
			return
		}

		var query = &pb.PokemonQuery{}
		if err := proto.Unmarshal(message, query); err != nil {
			fmt.Println("Error unmarshaling query:", err)
			errMsg, _ := marshalErrorMessage("unknow command query", 2)
			conn.send <- errMsg
			continue
		}

		if query.Id == "" && query.Name == "" && query.Region == "" {
			// If no query is specified, return the full Pokemon list
			pokemonListBytes, err := marshalPokemonList(pokemonList)
			if err != nil {
				errMsg, _ := marshalErrorMessage("failed to marshal PokemonList", 1)
				conn.send <- errMsg
				return
			}

			conn.send <- pokemonListBytes
		} else {
			// Filter the Pokemon list based on the query
			var results = &pb.PokemonList{}
			for _, p := range pokemonList.Pokemon {
				if (query.Id != "" && p.Id == query.Id) || (query.Name != "" && p.Name == query.Name) || (query.Region != "" && p.Region == query.Region) {
					results.Pokemon = append(results.Pokemon, p)
				}
			}

			pokemonListBytes, err := marshalPokemonList(results)
			if err != nil {
				errMsg, _ := marshalErrorMessage("failed to marshal PokemonList", 1)
				conn.send <- errMsg
				return
			}

			conn.send <- pokemonListBytes
		}
	}
}

func main() {
	// Create a map to hold the WebSocket connections
	connections := make(map[*Connection]bool)

	// Define a WebSocket upgrade handler
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// Define an HTTP handle to handle root path requests
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello, this is the Pokemon WebSocket server!")
	})

	// Define an HTTP handler function to handle WebSocket connections
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		// Upgrade the connection to a WebSocket connection
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("Error upgrading connection to WebSocket:", err)
			return
		}

		fmt.Println("New connection established", ws.RemoteAddr().String())

		// Handle the WebSocket connection
		handleConnection(ws, connections)
	})

	// Start the HTTP server
	log.Println("Starting server...")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("Error starting server:", err)
	}
}
