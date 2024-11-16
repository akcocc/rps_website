package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/a-h/templ"
	"github.com/gorilla/websocket"
)

var UPGRADER = websocket.Upgrader{}

// i dont wanna do proper error handling for this
func expect(err error) {
	if err != nil {
		err = fmt.Errorf("FATAL: %s", err)
		panic(err)
	}
}

func load_image_data(path string, resp_writer http.ResponseWriter) {
	img_data, err := os.ReadFile(path)
	expect(err)

	resp_writer.Header().Add("Content-Type", "image/svg+xml")
	n, err := resp_writer.Write(img_data)
	expect(err)

	fmt.Printf("%d bytes written\n", n)
}

type Room struct {
	players [2]*Client
}

type Hub struct {
	connections map[string]*Client
	rooms       [10]*Room
	register    chan struct {
		*Client
		string
	}
	unregister chan string
}

func new_hub() Hub {
	return Hub{
		connections: make(map[string]*Client),
		register: make(chan struct {
			*Client
			string
		}),
		unregister: make(chan string),
	}
}

func (hub *Hub) run() {
	for {
		select {
		case new_client := <-hub.register:
			hub.connections[new_client.string] = new_client.Client
		case client := <-hub.unregister:
			close(hub.connections[client].hub_message)
			delete(hub.connections, client)
		}
	}
}

type Client struct {
	connection  *websocket.Conn
	remote_addr string
	hub_message chan []byte
	hub         *Hub
}

type MessageType int
const (
    NameIntroduction = iota + 1
)

func server_websocket_connection(hub *Hub, resp_writer http.ResponseWriter, req *http.Request) {
	println("new address: ", req.RemoteAddr)
	connection, err := UPGRADER.Upgrade(resp_writer, req, nil)
	expect(err)
	defer connection.Close()
	remote_addr := connection.RemoteAddr().String()

	_, exists := hub.connections[remote_addr]
	if exists {
		println("connection exists")
	}

    for {
        message_type, message, err := connection.ReadMessage()
        if err != nil {
            if err.Error() != "websocket: close 1001 (going away)" {
                println("Websocket Error: ", err.Error())
            }
            println("Websocket connection closed for remote address: ", remote_addr)
            client.hub.unregister <- &client
            break
        }
        switch message_type {
        case websocket.TextMessage:
            name_data
            json.Unmarshal(message, )
            println(string(message))
        case websocket.PingMessage:
            println("Ping from remote address: ", remote_addr)
        }
        time.Sleep(1000 * time.Millisecond)
    }
}

func main() {
	home_component := home()

	server_hub := new_hub()

	go server_hub.run()

	http.Handle("/", templ.Handler(home_component))
	http.HandleFunc("/styles.css", func(resp_writer http.ResponseWriter, req *http.Request) {
		styles_data, err := os.ReadFile("styles.css")
		expect(err)

		resp_writer.Header().Add("Content-Type", "text/css")
		n, err := resp_writer.Write(styles_data)
		expect(err)

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

	http.HandleFunc("/game_round", func(resp_writer http.ResponseWriter, req *http.Request) {
		server_websocket_connection(&server_hub, resp_writer, req)
	})

	http.HandleFunc("/introduce", func(resp_writer http.ResponseWriter, req *http.Request) {
		req.ParseForm()
		client := Client{
			connection:  nil,
			hub_message: make(chan []byte),
			hub:         &server_hub,
		}
		client.hub.register <- struct {
			*Client
			string
		}{&client, req.RemoteAddr}
		player_name := req.PostForm.Get("player_name")
		println("player name: ", player_name)
		waiting_screen(player_name).Render(context.Background(), resp_writer)
	})

	http.HandleFunc("/scissors.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
		load_image_data("scissors-svgrepo-com.svg", resp_writer)
	})

	fmt.Println("Listening on port 8080")
	http.ListenAndServe(":8080", nil)
}
