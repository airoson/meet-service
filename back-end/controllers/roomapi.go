package controllers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"meet-service/sfu"
	"meet-service/utils"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type RoomInfo struct {
	Id          string `json:"id"`
	CreatedAt   string `json:"createdAt"`
	UsersCount  int    `json:"usersCount"`
	StartsAt    string `json:"startsAt,omitempty"`
	Description string `json:"description,omitempty"`
	Title       string `json:"title,omitempty"`
	Creator     string `json:"creator"`
	Active      bool   `json:"active"`
}

type RoomRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAt    string `json:"startsAt"`
}

func (restApi *RestApi) HandleRoomGetRequest(response http.ResponseWriter, request *http.Request) {
	roomIds := request.URL.Query()["roomId"]
	userIds := request.URL.Query()["userId"]
	if roomIds != nil && userIds != nil {
		response.WriteHeader(400)
		return
	}
	if roomIds != nil {
		roomId := roomIds[0]
		rows, err := restApi.Db.Query(
			`SELECT 
			created_at, 
			starts_at, 
			title, 
			description, 
			(SELECT COUNT(*) FROM user_at_call WHERE room_id=$1) as users_count, 
			shown_name as creator,
			active
		from room r JOIN registered_user ru ON r.owner_id = ru.user_id where room_id=$1;`, roomId)
		if err != nil {
			log.Println("Error: can't query room: ", err)
			response.WriteHeader(400)
			return
		}
		if rows.Next() {
			var (
				createdAt   time.Time
				startAt     sql.NullTime
				title       string
				description string
				usersCount  int
				active      bool
				creator     string
			)
			rows.Scan(&createdAt, &startAt, &title, &description, &usersCount, &creator, &active)
			room := RoomInfo{
				Id:          roomId,
				CreatedAt:   createdAt.Format(TIME_FORMAT),
				Description: description,
				Title:       title,
				UsersCount:  usersCount,
				Creator:     creator,
				Active:      active,
			}
			if startAt.Valid {
				room.StartsAt = startAt.Time.Format(TIME_FORMAT)
			}
			data, _ := json.Marshal(room)
			response.WriteHeader(200)
			response.Write(data)
		} else {
			response.WriteHeader(404)
			notFound := struct {
				Message string `json:"message"`
			}{fmt.Sprintf("room with id %s not found", roomId)}
			data, _ := json.Marshal(notFound)
			response.Write(data)
		}
	} else if userIds != nil {
		userId := userIds[0]
		rows, err := restApi.Db.Query(
			`SELECT 
			room_id,
			created_at, 
			starts_at, 
			title, 
			description, 
			(SELECT COUNT(*) FROM user_at_call WHERE room_id=r.room_id) as users_count,
			active
		from room r JOIN registered_user ru ON r.owner_id = ru.user_id where owner_id=$1;`, userId)
		if err != nil {
			log.Println("Error: can't query room: ", err)
			response.WriteHeader(400)
			return
		}
		results := []RoomInfo{}
		for rows.Next() {
			var (
				roomId      string
				createdAt   time.Time
				startsAt    sql.NullTime
				title       string
				description string
				userCount   int
				active      bool
			)
			rows.Scan(&roomId, &createdAt, &startsAt, &title, &description, &userCount, &active)
			info := RoomInfo{
				Id:          roomId,
				CreatedAt:   createdAt.Format(TIME_FORMAT),
				Description: description,
				Title:       title,
				UsersCount:  userCount,
				Active:      active,
			}
			if startsAt.Valid {
				info.StartsAt = startsAt.Time.Format(TIME_FORMAT)
			}
			results = append(results, info)
		}
		msg := struct {
			Data []RoomInfo `json:"data"`
		}{
			Data: results,
		}
		data, _ := json.Marshal(msg)
		response.WriteHeader(200)
		response.Write(data)
	} else {
		response.WriteHeader(400)
		return
	}
}

func (restApi *RestApi) HandleRoomPostRequest(response http.ResponseWriter, request *http.Request) {
	user, err := utils.ExtractAuthUserFromRequest(request)
	if err != nil {
		log.Printf("Error: %v\n", err)
		response.WriteHeader(400)
		return
	}
	req := RoomRequest{}
	data, err := io.ReadAll(request.Body)
	fatal(err)
	err = json.Unmarshal(data, &req)
	if err != nil {
		WriteMessage(response, "Wrong user data", 400)
		return
	}
	var id string
	isActive := true
	start := sql.NullTime{}
	if req.StartsAt != "" {
		log.Println("Planning room")
		start.Time, err = time.Parse(LOCAL_DATETIME, req.StartsAt)
		if err != nil {
			WriteMessage(response, "Wrong time formation", 400)
			return
		}
		start.Valid = true
		if start.Time.Compare(time.Now()) <= 0 {
			WriteMessage(response, "Start time is before current time", 400)
			return
		}
		id = uuid.New().String()
		isActive = false
	} else {
		log.Println("Starting room")
		id = restApi.Manager.CreateRoom()
	}
	_, err = restApi.Db.Exec(
		`INSERT INTO 
		room(room_id, created_at, starts_at, title, description, owner_id, active) 
		VALUES($1, $2, $3, $4, $5, $6, $7)`, id, time.Now(), start, req.Title, req.Description, user.UserId, isActive)
	if err != nil {
		log.Println("Can't create room: ", err)
		return
	}
	log.Println("User id is", user.UserId)
	rows, err := restApi.Db.Query("SELECT shown_name FROM registered_user WHERE user_id=$1;", user.UserId)
	fatal(err)
	if rows.Next() {
		var name string
		rows.Scan(&name)
		info := RoomInfo{
			Id:         id,
			CreatedAt:  time.Now().Format(TIME_FORMAT),
			UsersCount: 0,
			Creator:    name,
			Active:     isActive,
		}
		data, _ := json.Marshal(info)
		response.WriteHeader(200)
		response.Write(data)
		return
	}
	log.Println("User is authenticated but his account was deleted")
	response.WriteHeader(401)
}

func (restApi *RestApi) HandleRoomConnectionRequest(response http.ResponseWriter, request *http.Request) {
	user, err := utils.ExtractAuthUserFromRequest(request)
	anon := true
	var userId string
	if err == nil {
		userId = user.UserId
		anon = false
	}
	roomId := request.URL.Query()["roomId"][0]
	shownName := request.URL.Query()["shownName"][0]
	room := restApi.Manager.GetRoom(roomId)
	if room == nil {
		fmt.Fprintln(response, "room not found")
		return
	}
	conn, err := restApi.Upgrader.Upgrade(response, request, nil)
	if err != nil {
		log.Println("Can't establish websocket connection: ", err)
		return
	}
	onExit := func() {
		_, err := restApi.Db.Exec("DELETE FROM user_at_call WHERE user_id=$1", userId)
		if err != nil {
			log.Println("Error while deleting user from room:", err)
		}
		rows, err := restApi.Db.Query("SELECT count(*) FROM user_at_call WHERE room_id=$1", roomId)
		if err != nil {
			log.Println("Can't get number of users in room")
		} else {
			rows.Next()
			var count int
			err = rows.Scan(&count)
			rows.Close()
			if err == nil && count == 0 {
				_, err = restApi.Db.Exec("UPDATE room SET last_interact=$1 WHERE room_id=$2", time.Now(), roomId)
				if err != nil {
					log.Println("Can't update room last_interact value")
				}
			}
		}
		err = conn.Close()
		if err != nil {
			log.Println("Can't close websocket connection: ", err)
		}
	}
	if anon {
		userId, err = room.CreateAndAddUser(onExit)
		anon = true
	} else {
		err = room.AddUser(userId, onExit)
	}
	if err != nil {
		conn.Close()
		log.Println("Can't create user: ", err)
		return
	}
	log.Printf("Crete new user in room %s - %s (anon: %v)\n", roomId, userId, anon)
	//Add info about user in the user_at_call table
	admin := false
	rows, err := restApi.Db.Query("SELECT call_start, owner_id from room WHERE room_id=$1;", roomId)
	fatal(err)
	if !rows.Next() {
		//Room was deleted
		WriteMessage(response, "Room was deleted", 400)
		log.Println("Attempt to connect to room that was deleted, but still available in the room manager")
		restApi.Manager.DeleteRoom(roomId)
		return
	}
	var owner string
	var callStart sql.NullTime
	rows.Scan(&callStart, &owner)
	rows.Close()
	if owner == userId || !anon && user.Role == "ADMIN" {
		admin = true
	}
	var start time.Time
	if !callStart.Valid {
		start = time.Now()
		_, err = restApi.Db.Exec("UPDATE room SET call_start=$1 where room_id=$2", start, roomId)
		if err != nil {
			log.Println("Can't update call_start for room ", roomId)
		}
	} else {
		start = callStart.Time
	}
	callStartMsg := struct {
		CallStart string `json:"callStart"`
	}{start.Format(TIME_FORMAT)}
	data, _ := json.Marshal(callStartMsg)
	err = conn.WriteMessage(websocket.TextMessage, data)
	if err != nil {
		log.Println("can't send callStart websocket message: ", err)

		return
	}

	_, err = restApi.Db.Exec("INSERT INTO user_at_call(user_id, room_id, shown_name, is_admin, anon) values ($1, $2, $3, $4, $5)",
		userId, roomId, shownName, admin, anon)
	if err != nil {
		log.Println("Can't save meet user to database: ", err)
		conn.Close()
		return
	}
	go func() {
		for {
			messageType, p, err := conn.ReadMessage()
			log.Println("Receive message from user ", userId)
			if err != nil {
				room.DeleteUser(userId)
				log.Println("Error while receiving message from websocket connection: ", err)
				return
			}
			if messageType == websocket.TextMessage {
				msg := sfu.Message{}
				err = json.Unmarshal(p, &msg)
				if err != nil {
					log.Println("Receive invalid data: ", err)
					continue
				}
				err = room.ProcessMessage(userId, msg)
				if err != nil {
					log.Println("Error while processing message: ", err)
					return
				}
			}
		}
	}()

	//Read message channel and send back messages to user
	go func() {
		for {
			msg := room.ReceiveMessage(userId)
			log.Println("Sending back messages ro user ", userId)
			if msg.StreamMapping != "" {
				otherId := strings.Split(msg.StreamMapping, ":")[1]
				rows, err := restApi.Db.Query("SELECT shown_name FROM user_at_call WHERE user_id = $1", otherId)
				var name string
				if err != nil || !rows.Next() {
					log.Println("Can't find shown name for user ", otherId)
					name = otherId
				} else {
					rows.Scan(&name)
				}
				msg.StreamMapping += ":" + name
			}
			data, _ := json.Marshal(msg)
			err := conn.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				log.Println("Error while sending back message to socket: ", err)
				return
			}
		}
	}()
}

func (restApi *RestApi) HandleRoomDeleteRequest(response http.ResponseWriter, request *http.Request) {
	user, err := utils.ExtractAuthUserFromRequest(request)
	if err != nil {
		response.WriteHeader(401)
		return
	}
	roomIds := request.URL.Query()["roomId"]
	if roomIds != nil {
		roomId := roomIds[0]
		room := restApi.Manager.GetRoom(roomId)
		if room == nil {
			WriteMessage(response, "Room not found", 404)
			return
		}
		rows, err := restApi.Db.Query("SELECT owner_id FROM room WHERE room_id = $1", roomId)
		fatal(err)
		if !rows.Next() {
			log.Println("Error: room presents in room manager but doesn't saved in database")
			room.DeleteRoom()
			response.WriteHeader(200)
			return
		}
		var owner string
		rows.Scan(&owner)
		if owner != user.UserId {
			response.WriteHeader(401)
			return
		}
		_, err = restApi.Db.Exec("DELETE FROM room WHERE room_id = $1", roomId)
		if err != nil {
			log.Println("Error while deleting room from db: ", err)
		}
		room.DeleteRoom()
		response.WriteHeader(200)
	} else {
		response.WriteHeader(400)
		return
	}
}
