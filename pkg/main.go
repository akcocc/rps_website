package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"rps_website/pkg/assert"
	"rps_website/pkg/hub"
	"strings"

	"github.com/a-h/templ"
	"github.com/gorilla/websocket"
)

var UPGRADER = websocket.Upgrader{}


func load_image_data(path string, resp_writer http.ResponseWriter) {
    img_data, err := os.ReadFile(path)
    assert.Expect(err, "failed to read image file")

    resp_writer.Header().Add("Content-Type", "image/svg+xml")
    _, err = resp_writer.Write(img_data)
    assert.Expect(err, "failed to write image data into response writer")

    // fmt.Printf("%d bytes written\n", n)
}


func serve_websocket_connection(h *hub.Hub, resp_writer http.ResponseWriter, req *http.Request) {
    connection, err := UPGRADER.Upgrade(resp_writer, req, nil)
    assert.Expect(err, "could not upgrade client connection to websocket")
    defer connection.Close()
    println("wsoc connection established")

    assert.Assert(connection != nil, "client connection shouldn't be nil")
    writer, err := connection.NextWriter(websocket.TextMessage)
    assert.Expect(err, "could not get writer for next message")
    greeting_screen().Render(context.Background(), writer)
    writer.Close()

    assert.Assert(h != nil, "hub shouldn't be nil")

    hub.Handle_client(connection, h)
}

func main() {
    home_component := home()

    server_hub := hub.New_hub()

    go server_hub.Run()

    http.Handle("/", templ.Handler(home_component))
    http.HandleFunc("/styles.css", func (resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        styles_data, err := os.ReadFile("styles.css")
        assert.Expect(err, "failed to read styles file")

        resp_writer.Header().Add("Content-Type", "text/css")
        _, err = resp_writer.Write(styles_data)
        assert.Expect(err, "failed to write styles data into response writer")

        // fmt.Printf("%d bytes written\n", n)
    })

    http.HandleFunc("/rock.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        load_image_data("rock-svgrepo-com.svg", resp_writer)
    })

    http.HandleFunc("/spinner.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        load_image_data("833.svg", resp_writer)
    })

    http.HandleFunc("/action/", func(resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        parts := strings.Split(req.RequestURI, "action/")
        fmt.Println(parts[1])
    })

    http.HandleFunc("/paper.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        load_image_data("paper-document-file-data-svgrepo-com.svg", resp_writer)
    })

    http.HandleFunc("/connect", func(resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        serve_websocket_connection(&server_hub, resp_writer, req)
    })

    http.HandleFunc("/scissors.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        assert.Assert(req != nil, "request should not be nil")
        load_image_data("scissors-svgrepo-com.svg", resp_writer)
    })

    fmt.Println("Listening on port 443")
    err := http.ListenAndServeTLS(":443", "server.crt", "server.key", nil)
    assert.Expect(err, "could not start server")
}
