package controllers

import (
	"io"
	"log"
	"net/http"
	"os"
)

type HTMLHandler struct {
	PathToFile string
}

func (handler *HTMLHandler) ServeHTTP(response http.ResponseWriter, request *http.Request) {
	if request.Method != "GET" {
		response.WriteHeader(405)
		return
	}
	log.Println("Request for file ", request.URL.Path)
	file, err := os.Open(handler.PathToFile)
	if err != nil {
		response.WriteHeader(401)
		log.Println("Error while access the file: ", err)
		return
	}
	response.WriteHeader(200)
	response.Header().Add("content-type", "text/html")
	io.Copy(response, file)
}
