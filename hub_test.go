package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"rps_website/assert"
	"rps_website/hub"
	"testing"

	"github.com/gorilla/websocket"
)

func TestHub(t *testing.T) {
	go run_server()

	go func() {
		conn1 := new_client()
		defer conn1.Close()

		greet_screen_test(conn1, t)
	}()

	go func() {
		conn2 := new_client()
		defer conn2.Close()

		greet_screen_test(conn2, t)
	}()
}

func name_send_test(conn *websocket.Conn, t *testing.T) {
}

func greet_screen_test(conn *websocket.Conn, t *testing.T) {
	var greet_screen hub.Buf
	greeting_screen().Render(context.Background(), &greet_screen)
	msg_type, msg, err := conn.ReadMessage()
	if err != nil {
		assert.Expect(err, "error when reading message from server")
	}

	if msg_type != websocket.TextMessage {
		fmt.Printf("Assertion failed: expected message type of %d, found %d", websocket.TextMessage, msg_type)
		t.FailNow()
	}

	if string(msg) != string(greet_screen) {
		fmt.Printf("Assertion failed: expected greering screen of format:\n%s\n\ngot:\n%s", string(greet_screen), string(msg))
		t.FailNow()
	}
}

func new_client() *websocket.Conn {
	url := url.URL{
		Scheme:      "ws",
		Host:        "localhost:4443",
		Path:        "/connect",
	}
	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		assert.Expect(err, "Client could not connect")
	}
	return conn
}

func run_server() {
	hub := hub.New_hub()
	go hub.Run()

	http.HandleFunc("/connect", func(resp_writer http.ResponseWriter, req *http.Request) {
		serve_websocket_connection(&hub, resp_writer, req)
	})

	fmt.Println("Listening on port ", LISTENING_PORT)
	err := http.ListenAndServeTLS(fmt.Sprintf(":%d", LISTENING_PORT), "toopsi.dev.pem", "toopsi.dev.key", nil)
	assert.Expect(err, "could not start server")
}
