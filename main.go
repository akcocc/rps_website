package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/gorilla/websocket"
)

var UPGRADER = websocket.Upgrader{}

// i dont wanna do proper error handling for this
func expect(err error, message string) {
    if err != nil {
        err = fmt.Errorf("FATAL: %s: %s", message, err)
        panic(err)
    }
}

func load_image_data(path string, resp_writer http.ResponseWriter) {
    img_data, err := os.ReadFile(path); expect(err, "failed to read image file")

    resp_writer.Header().Add("Content-Type", "image/svg+xml")
    n, err := resp_writer.Write(img_data); expect(err, "failed to write image data into response writer")

    fmt.Printf("%d bytes written\n", n)
}

type Room struct {
    players [2]*Client
}

type Hub struct {
    connections map[*Client]bool
    rooms [3]*Room
    register chan *Client
    unregister chan *Client
}

func new_hub() Hub {
    hub := Hub{
    	connections: make(map[*Client]bool),
    	rooms:       [3]*Room{},
    	register:    make(chan *Client),
    	unregister:  make(chan *Client),
    }
    hub.make_rooms()
    return hub
}

func (hub *Hub) make_rooms() {
    for i := range hub.rooms {
        hub.rooms[i] = new_room()
    }
}

func new_room() *Room {
    return &Room{
    	players: [2]*Client{},
    }
}

func (hub *Hub) run() {
    for {
        select {
            case client := <-hub.register:
                hub.connections[client] = true
            case client := <-hub.unregister:
                println("unregistering cient")
                hub.search_and_remove_player(client)
                delete(hub.connections, client)
                close(client.hub_message)
        }
    }
}

func (hub *Hub) search_and_remove_player(player *Client) {
    for room_number, room := range hub.rooms {
        for spot_number := range room.players {
            if room.players[spot_number] != nil && room.players[spot_number].id == player.id {
                println("found spot")
                room.players[spot_number] = nil
                fmt.Printf("Removed player from Room #%d, Spot #%d\n", room_number, spot_number)

                // tell other player, if they exist, that the other player has disconnected
                // then send them back to the home screen
                return
            }
        }
    }
}

func (hub *Hub) search_for_room() (int, int) {
    for room_number, room := range hub.rooms {
        for spot_number, player := range room.players {
            if player == nil {
                return room_number, spot_number
            }
        }
    }
    return -1, -1
}

type Client struct {
    connection *websocket.Conn
    player_name string
    id [16]byte
    hub_message chan []byte
    hub *Hub
}

type Action int
const (
    ROCK = iota + 1
    PAPER
    SCISSORS
)

func assert(predicate bool, message string) {
    if !predicate {
        panic(message)
    }
}

func assert_eq(item1 interface{}, item2 interface{}, message string) {
    if item1 != item2 {
        panic(fmt.Sprintf("item 1: %v, item 2: %v: %s", item1, item2, message))
    }
}

func serve_websocket_connection(hub *Hub, resp_writer http.ResponseWriter, req *http.Request) {
    connection, err := UPGRADER.Upgrade(resp_writer, req, nil); expect(err, "could not upgrade client connection to websocket")
    defer connection.Close()
    println("player connected")
    remote_addr := connection.RemoteAddr().String()
    client := Client{
    	connection:  connection,
    	hub_message: make(chan []byte),
    	hub:         hub,
    }

    rand.Read(client.id[:])

    client.hub.register <- &client

    message_type, message, err := client.connection.ReadMessage()
    if err != nil {
        close_client_connection(client, remote_addr, err)
        return
    }
    var filtered_message []byte = nil
    for filtered_message == nil {
        filtered_message = handle_client_message(message, message_type, remote_addr)
    }
    println(string(filtered_message))

    var data map[string]interface{}

    json.Unmarshal(message, &data); expect(err, "could not parse message data as json")

    client.player_name = data["player_name"].(string)
    writer, err := client.connection.NextWriter(websocket.TextMessage); expect(err, "could not get writer for next message")
    waiting_for_room(client.player_name).Render(context.Background(), writer)
    writer.Close()

    // place player in room_number
    var room_number, player_spot int = -1, -1
    for player_spot == -1 {
        room_number, player_spot = client.hub.search_for_room();
        if player_spot != -1 {
            break
        }
        println("waiting for room")
        time.Sleep(3 * time.Second)
    }

    fmt.Printf("Player placed in Room #%d, Spot #%d\n", room_number, player_spot)
    client.hub.rooms[room_number].players[player_spot] = &client

    writer, err = client.connection.NextWriter(websocket.TextMessage); expect(err, "could not get writer for next message")
    waiting_for_player(client.player_name).Render(context.Background(), writer)
    writer.Close()


    for {
        message_type, message, err := client.connection.ReadMessage()
        if err != nil {
            close_client_connection(client, remote_addr, err)
            break
        }
        message = handle_client_message(message, message_type, remote_addr)
        if message != nil {
            json.Unmarshal(message, &data); expect(err, "could not parse message data as json")
            player_action := data["action"].(Action)
            assert(1 <= player_action && player_action <= 3, "message should be between 1 and 3, inclusively")
        }
        time.Sleep(1000 * time.Millisecond)
    }
}

func handle_client_message(message []byte, message_type int, remote_addr string) []byte {
    switch message_type {
    case websocket.TextMessage:
        return message
    case websocket.PingMessage:
        println("Ping from remote address: ", remote_addr)
    }
    return nil
}

func close_client_connection(client Client, remote_addr string, err error) {
    if err.Error() != "websocket: close 1001 (going away)" {
        println("Websocket Error: ", err.Error())
    }
    println("Websocket connection closed for remote address: ", remote_addr)
    client.hub.unregister <- &client
}

func main() {
    home_component := home()

    server_hub := new_hub()

    go server_hub.run()

    http.Handle("/", templ.Handler(home_component))
    http.HandleFunc("/styles.css", func (resp_writer http.ResponseWriter, req *http.Request) {
        styles_data, err := os.ReadFile("styles.css"); expect(err, "failed to read styles file")
        fmt.Println(req.RemoteAddr)

        resp_writer.Header().Add("Content-Type", "text/css")
        n, err := resp_writer.Write(styles_data); expect(err, "failed to write styles data into response writer")

        fmt.Printf("%d bytes written\n", n)
    })

    http.HandleFunc("/rock.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        load_image_data("rock-svgrepo-com.svg", resp_writer)
    })

    http.HandleFunc("/action/", func(resp_writer http.ResponseWriter, req *http.Request) {
        parts := strings.Split(req.RequestURI, "action/")
        fmt.Println(parts[1])
    })

    http.HandleFunc("/paper.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        load_image_data("paper-document-file-data-svgrepo-com.svg", resp_writer)
    })

    http.HandleFunc("/connect", func(resp_writer http.ResponseWriter, req *http.Request) {
        serve_websocket_connection(&server_hub, resp_writer, req)
    })

    http.HandleFunc("/scissors.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        load_image_data("scissors-svgrepo-com.svg", resp_writer)
    })

    fmt.Println("Listening on port 443")
    err := http.ListenAndServeTLS(":443", "server.crt", "server.key", nil); expect(err, "could not start server")
}
