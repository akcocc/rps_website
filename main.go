package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/a-h/templ"
)

// i dont wanna do proper error handling for this
func expect(err error) {
    if err != nil {
        err = fmt.Errorf("FATAL: %s", err)
        panic(err)
    }
}

func load_image_data(path string, resp_writer http.ResponseWriter) {
    img_data, err := os.ReadFile(path); expect(err)

    resp_writer.Header().Add("Content-Type", "image/svg+xml")
    n, err := resp_writer.Write(img_data); expect(err)

    fmt.Printf("%d bytes written\n", n)
}

func main() {
    component := home("Zach")

    http.Handle("/", templ.Handler(component))
    http.HandleFunc("/styles.css", func (resp_writer http.ResponseWriter, req *http.Request) {
        styles_data, err := os.ReadFile("styles.css"); expect(err)
        fmt.Println(req.RemoteAddr)

        resp_writer.Header().Add("Content-Type", "text/css")
        n, err := resp_writer.Write(styles_data); expect(err)

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

    http.HandleFunc("/scissors.svg", func(resp_writer http.ResponseWriter, req *http.Request) {
        load_image_data("scissors-svgrepo-com.svg", resp_writer)
    })

    fmt.Println("Listening on port 8080")
    http.ListenAndServe(":8080", nil)
}
