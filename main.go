package main

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"os"
	"rps_website/assert"
	"rps_website/hub"
	"strings"

	"github.com/gorilla/websocket"
)

var UPGRADER = websocket.Upgrader{}

const LISTENING_PORT = 4443

func load_image_data(path string, resp_writer http.ResponseWriter) {
	img_data, err := os.ReadFile(path)
	if err != nil {
		if strings.Contains(err.Error(), "no such file or directory") {
			resp_writer.WriteHeader(404)
			return
		}
		assert.Expect(err, "failed to read image file")
	}

	digest := md5.Sum(img_data)

	resp_writer.Header().Add("Cache-Control", "public, max-age=31536000, immutable")
	resp_writer.Header().Add("Etag", fmt.Sprintf("%x", digest))
	_, err = resp_writer.Write(img_data)
	assert.Expect(err, "failed to write image data into response writer")
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

func setup_endpoint_handlers(serve_mux *http.ServeMux, server_hub *hub.Hub) {
	serve_mux.HandleFunc("/", func(resp_writer http.ResponseWriter, req *http.Request) {
		println("bar")
		resp_writer.Header().Add("Content-Type", "text/html")
		resp_writer.Header().Add("Cache-Control", "public, max-age=31536000, immutable")
		var buf hub.Buf
		if req.URL.Path != "/" {
			resp_writer.WriteHeader(404)
			resp_writer.Header().Add("Connection", "close")
			not_found().Render(req.Context(), &buf)
		} else if server_hub.Is_full() {
			resp_writer.WriteHeader(503)
			resp_writer.Header().Add("Connection", "close")
			unavailable().Render(req.Context(), &buf)
		} else {
			home().Render(req.Context(), &buf)
		}
		digest := md5.Sum(buf)
		resp_writer.Header().Add("Etag", fmt.Sprintf("%x", digest))
		resp_writer.Write(buf)
	})

	// serve_mux.HandleFunc("/", func(resp_writer http.ResponseWriter, req *http.Request) {
	// 	println("foo")
	// 	resp_writer.Header().Add("Content-Type", "text/html")
	// 	resp_writer.Header().Add("Cache-Control", "public, max-age=31536000, immutable")
	// 	var buf hub.Buf
	// 	if req.URL.Path != "/" {
	// 		resp_writer.WriteHeader(404)
	// 		println("baz")
	// 		resp_writer.Header().Add("Connection", "close")
	// 		not_found().Render(req.Context(), &buf)
	// 	} else if server_hub.Is_full() {
	// 		resp_writer.WriteHeader(503)
	// 		resp_writer.Header().Add("Connection", "close")
	// 		unavailable().Render(req.Context(), &buf)
	// 	} else {
	// 		home().Render(req.Context(), &buf)
	// 	}
	// 	digest := md5.Sum(buf)
	// 	resp_writer.Header().Add("Etag", fmt.Sprintf("%x", digest))
	// 	resp_writer.Write(buf)
	// })

	serve_mux.HandleFunc("/connect", func(resp_writer http.ResponseWriter, req *http.Request) {
		serve_websocket_connection(server_hub, resp_writer, req)
	})
	// serve_mux.HandleFunc("/back_to_hub", func(resp_writer http.ResponseWriter, req *http.Request) {
	// 	resp_writer.Header().Add("Content-Type", "text/html")
	// 	greeting_screen().Render(req.Context(), resp_writer)
	// })
	serve_mux.HandleFunc("/styles.css", func(resp_writer http.ResponseWriter, req *http.Request) {
		assert.Assert(req != nil, "request should not be nil")
		styles_data, err := os.ReadFile("styles.css")
		assert.Expect(err, "failed to read styles file")

		digest := md5.Sum(styles_data)

		resp_writer.Header().Add("Content-Type", "text/css")
		resp_writer.Header().Add("Cache-Control", "public, max-age=31536000, immutable")
		resp_writer.Header().Add("Etag", fmt.Sprintf("%x", digest))
		_, err = resp_writer.Write(styles_data)
		assert.Expect(err, "failed to write styles data into response writer")

		// fmt.Printf("%d bytes written\n", n)
	})

	serve_mux.HandleFunc("/jpg/{image}", func(resp_writer http.ResponseWriter, req *http.Request) {
		image_name := req.PathValue("image")
		if image_name[(len(image_name)-4):] != ".jpg" {
			println(image_name)
			resp_writer.WriteHeader(404)
			return
		}
		assert.Assert(req != nil, "request should not be nil")
		load_image_data(image_name, resp_writer)
	})

	serve_mux.HandleFunc("/svg/{image}", func(resp_writer http.ResponseWriter, req *http.Request) {
		image_name := req.PathValue("image")
		if image_name[(len(image_name)-4):] != ".svg" {
			println(image_name)
			resp_writer.WriteHeader(404)
			return
		}
		assert.Assert(req != nil, "request should not be nil")
		resp_writer.Header().Add("Content-Type", "image/jpg")
		resp_writer.Header().Add("Content-Type", "image/svg+xml")
		load_image_data(image_name, resp_writer)
	})
}

func main() {
	server_hub := hub.New_hub()

	go server_hub.Run()

	serve_mux := http.NewServeMux()

	setup_endpoint_handlers(serve_mux, &server_hub)

	fmt.Println("Listening on port ", LISTENING_PORT)
	err := http.ListenAndServeTLS(fmt.Sprintf(":%d", LISTENING_PORT), "toopsi.dev.pem", "toopsi.dev.key", serve_mux)
	assert.Expect(err, "could not start server")
}
