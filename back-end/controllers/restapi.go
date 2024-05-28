package controllers

import (
	"database/sql"
	"encoding/json"
	"meet-service/sfu"
	"net/http"

	"github.com/gorilla/websocket"
)

type RestApi struct {
	Manager  sfu.RoomManager
	Db       *sql.DB
	Upgrader websocket.Upgrader
}

const TIME_FORMAT = "02.01.2006 15:04:05"
const LOCAL_DATETIME = "2006-01-02T15:04"

func WriteMessage(response http.ResponseWriter, message string, status int) {
	response.WriteHeader(status)
	msg := struct {
		Message string `json:"message"`
	}{message}
	data, _ := json.Marshal(msg)
	response.Write(data)
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}
