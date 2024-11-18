package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"rps_website/pkg/assert"
	"time"

	"github.com/gorilla/websocket"
)

type Hub struct {
    connections map[*Client]bool
    rooms [3]*Room
    register chan *Client
    unregister chan *Client
    connection_count uint
}

type Client struct {
    connection *websocket.Conn
    player_name string
    id uint
    message_channel chan []byte
    room_number int
    in_match bool
    hub *Hub
}

func New_hub() Hub {
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

func (hub *Hub) Run() {
    for {
        select {
            case client := <-hub.register:
                if client != nil {
                    println("registering client")
                    hub.connections[client] = true
                } else {
                    println("ERROR: Couldn't register client, client pointer was nil")
                }
            case client := <-hub.unregister:
                println("unregistering client")
                if client != nil{
                    hub.search_and_remove_player(client)
                } else {
                    println("ERROR: Couldn't unregister client, client pointer was nil")
                }
                delete(hub.connections, client)
        }
    }
}

func close_client_connection(client Client, err error) {
    if err.Error() != "websocket: close 1001 (going away)" {
        println("Websocket Error: ", err.Error())
    }
    assert.Assert(client.id != 0, "client id shouldnt be 0")
    println("Websocket connection closed for client id: ", client.id)

    // if client is in a room with another player
    if client.in_match {
        client.message_channel <- []byte("left")

        // cleanup
        client.in_match = false
    }
    client.hub.unregister <- &client
}

func handle_client_message(message []byte, message_type int) []byte {
    if message_type == websocket.TextMessage {
        return message
    }
    return nil
}

func new_client(connection *websocket.Conn, hub *Hub) Client {

    client := Client {
        connection:  connection,
        message_channel: make(chan []byte),
        room_number: -1,
        hub:         hub,
    }

    return client
}

func (client *Client) get_player_name() error {
    assert.Assert(client.connection != nil, "client connection shouldn't be nil")
    message_type, message, err := client.connection.ReadMessage()
    if err != nil {
        return err
    }
    var filtered_message []byte = nil
    for filtered_message == nil {
        filtered_message = handle_client_message(message, message_type)
    }
    println(string(filtered_message))

    var data map[string]interface{}

    err = json.Unmarshal(message, &data)
    assert.Expect(err, "could not parse message data as json")

    assert.Assert(&client != nil, "client should not be nil")
    client.player_name = data["player_name"].(string)
    return nil
}

func (client *Client) send_wait_room_screen() {
    writer, err := client.connection.NextWriter(websocket.TextMessage)
    defer writer.Close()
    assert.Expect(err, "could not get writer for next message")
    waiting_room_screen(client.player_name).Render(context.Background(), writer)
}

func (client *Client) send_player_left_screen(departed_player string) {
    writer, err := client.connection.NextWriter(websocket.TextMessage)
    defer writer.Close()
    assert.Expect(err, "could not get writer for next message")
    player_left(departed_player).Render(context.Background(), writer)
}

func (client *Client) send_wait_player_screen() {
    assert.Assert(client.connection != nil, "client connection shouldn't be nil")

    writer, err := client.connection.NextWriter(websocket.TextMessage)
    assert.Expect(err, "could not get writer for next message")

    waiting_for_player(client.player_name).Render(context.Background(), writer)
    writer.Close()
}

func (client *Client) wait_for_available_room() {
    var room_number, player_spot int = -1, -1
    for player_spot == -1 {
        room_number, player_spot = client.hub.search_for_room();
        if player_spot != -1 {
            break
        }
        println("waiting for room")
        time.Sleep(3 * time.Second)
    }
    // place player in room_number
    assert.Assert(client.hub != nil, "client.hub shouldn't be nil")
    client.hub.rooms[room_number].players[player_spot] = client
    client.room_number = room_number
    fmt.Printf("Player placed in Room #%d, Spot #%d\n", room_number + 1, player_spot + 1)
}

type Action int
const (
    ROCK = iota + 1
    PAPER
    SCISSORS
)

func Handle_client(connection *websocket.Conn, hub *Hub) {
    client := new_client(connection, hub)
    client.hub.register <- &client

    hub.connection_count++
    client.id = hub.connection_count

    err := client.get_player_name()
    if err != nil {
        close_client_connection(client, err)
        return
    }

    client.send_wait_room_screen()

    client.wait_for_available_room()

    room := client.hub.rooms[client.room_number]
    if room.players[0] != nil && room.players[1] != nil {
        go room.mediate()
    }

    client.send_wait_player_screen()

    client_message_chan := make(chan struct { mt int; m []byte; e error})
    client_message := func() {
        message_type, message, err := client.connection.ReadMessage()
        client_message_chan<-struct{mt int; m []byte; e error}{
            message_type,
            message,
            err,
        }
    }

    var client_message_ready bool = true

    for {
        assert.Assert(client.connection != nil, "client connection shouldn't be nil")

        // initiates read on next client message

        if client_message_ready {
            go client_message()
            client_message_ready = false
        }

        select {

            // selects on client_message_chan to get new client messages
            case inc_message := <-client_message_chan:
                message_type, message, err := inc_message.mt, inc_message.m, inc_message.e
                if err != nil {
                    close_client_connection(client, err)
                    break
                }
                message = handle_client_message(message, message_type)
                var data map[string]interface{}
                if message != nil {
                    err = json.Unmarshal(message, &data)
                    player_action_raw := data["action"].(string)
                    client.message_channel<-[]byte(player_action_raw)

                    assert.Expect(err, "action should be able to parse to an integer")
                    // assert.Assert(1 <= player_action && player_action <= 3, "message should be between 1 and 3, inclusively")
                }
                client_message_ready = true


            // send message to client
            case room_message := <-client.message_channel:
                assert.Assert(client.connection != nil, "client connection shouldn't be nil")
                writer, err := client.connection.NextWriter(websocket.TextMessage)
                if err != nil {
                    close_client_connection(client, err)
                    break
                }
                writer.Write(room_message)
                writer.Close()
                client.message_channel<-[]byte("sent")
        }
    }
}

func (hub *Hub) search_and_remove_player(player *Client) {
    for room_number, room := range hub.rooms {
        for spot_number := range room.players {
            assert.Assert(player != nil, "player was nil")
            if room.players[spot_number] != nil && room.players[spot_number].id == player.id {
                room.players[spot_number] = nil
                fmt.Printf("Removed player from Room #%d, Spot #%d\n", room_number + 1, spot_number + 1)

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
