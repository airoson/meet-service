package sfu

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pion/webrtc/v4"
)

type Room struct {
	id        string
	users     map[string]*User
	mutex     *sync.RWMutex
	createdAt time.Time
}

type RoomInterface interface {
	CreateAndAddUser() (string, error)
	AddUser(userId string) error
	DeleteUser(id string)
	DeleteRoom()
	ProcessMessage(id string, msg Message)
	ReceiveMessage(id string) Message
	AddTrackToRoom(id string, track *webrtc.TrackRemote)
	UserCount() int
	CreatedAt() time.Time
}

type Message struct {
	MessageType      string `json:"type"`
	Sdp              string `json:"sdp"`
	Candidate        string `json:"candidate"`
	SdpMid           string `json:"sdpMid"`
	SdpMLineIndex    uint16 `json:"sdpMLineIndex"`
	UsernameFragment string `json:"usernameFragment"`
	StreamMapping    string `json:"streamMapping"`
	UsersCount       int    `json:"usersCount"`
}

func CreateRoom(id string) *Room {
	r := Room{
		id:        id,
		users:     map[string]*User{},
		mutex:     &sync.RWMutex{},
		createdAt: time.Now(),
	}
	return &r
}

func (room *Room) CreateAndAddUser(onExit func()) (string, error) {
	room.mutex.Lock()
	defer room.mutex.Unlock()
	user, err := CreateUser(uuid.New().String(), room.AddTrackToRoom, onExit)
	if err != nil {
		log.Println("Can't create user: ", err)
		return "", fmt.Errorf("can't create user: %v", err)
	}
	room.users[user.id] = user
	log.Println("Add user ", user.id)
	go room.SendRoomStateUpdate()
	return user.id, nil
}

func (room *Room) AddUser(userId string, onExit func()) error {
	room.mutex.Lock()
	defer room.mutex.Unlock()
	user, err := CreateUser(userId, room.AddTrackToRoom, onExit)
	if err != nil {
		log.Println("Can't create user: ", err)
		return fmt.Errorf("can't create user: %v", err)
	}
	room.users[user.id] = user
	log.Println("Add user ", user.id)
	go room.SendRoomStateUpdate()
	return nil
}

func (room *Room) GetMidMap(id string) map[string]string {
	user, ok := room.users[id]
	if !ok {
		log.Println("Attempt to get mid map for user that doesn't exist")
		return map[string]string{}
	}
	return user.midMap
}

func (room *Room) ReceiveMessage(id string) Message {
	user := room.users[id]
	if user == nil {
		log.Println("Trying to receive message from user that don't exists")
		return Message{}
	}
	msg, ok := <-user.msgChan
	if ok {
		return msg
	} else {
		return Message{}
	}
}

func (room *Room) ProcessMessage(id string, msg Message) error {
	user := room.users[id]
	if user == nil {
		return fmt.Errorf("trying to send message to user that don't exists")
	}
	if msg.Sdp != "" {
		if msg.MessageType == "offer" {
			log.Println("Receiving sdp offer")
			offer := msg.Sdp
			ans, err := room.users[id].ReceiveOffer(offer)
			if err != nil {
				log.Println("Can't process offer: ", err)
				return fmt.Errorf("can't process offer: %v", err)
			}
			outMsg := Message{}
			outMsg.Sdp = ans.SDP
			outMsg.MessageType = ans.Type.String()
			log.Println("Sending sdp answer")
			user.msgChan <- outMsg
			room.AddAllTracksToUser(id)
		} else if msg.MessageType == "answer" {
			log.Println("Receiving sdp answer")
			answer := msg.Sdp
			err := room.users[id].ReceiveAnswer(answer)
			if err != nil {
				log.Println("Can't process answer: ", err)
				return fmt.Errorf("can't process answer: %v", err)
			}
		}
	} else if msg.Candidate != "" {
		log.Println("Receiving ice candidate")
		candidate := webrtc.ICECandidateInit{
			Candidate:        msg.Candidate,
			SDPMid:           &msg.SdpMid,
			SDPMLineIndex:    &msg.SdpMLineIndex,
			UsernameFragment: &msg.UsernameFragment,
		}
		room.users[id].AddIceCandidate(candidate)
	} else {
		return fmt.Errorf("no sdp or ice candidate presented")
	}
	return nil
}

func (room *Room) AddAllTracksToUser(id string) {
	room.mutex.RLock()
	defer room.mutex.RUnlock()
	updatedUser := room.users[id]
	for userId, user := range room.users {
		if userId != id {
			for _, track := range user.localTracks {
				updatedUser.AddTrack(track, userId)
				log.Printf("[AddAllTracksToUser] Add track with id %s to user %s\n", track.ID(), id)
			}
		}
	}
}

func (room *Room) AddTrackToRoom(id string, track *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error) {
	room.mutex.Lock()
	defer room.mutex.Unlock()
	trackLocal, err := webrtc.NewTrackLocalStaticRTP(track.Codec().RTPCodecCapability, track.ID(), track.StreamID())
	if err != nil {
		log.Println("Can't add track to room:", err)
		return nil, fmt.Errorf("can't add track to room: %v", err)
	}
	for userId, user := range room.users {
		if userId != id {
			user.AddTrack(trackLocal, id)
			log.Printf("[AddTrackToRoom] Add track with id %s to user %s\n", trackLocal.ID(), userId)
		}
	}
	log.Printf("FULL TRACKS DESC: trackLocal id: %s, trackRemote id: %s, trackLocal RID: %s from user %s\n", trackLocal.ID(), track.ID(), trackLocal.RID(), id)
	return trackLocal, nil
}

func (room *Room) CreatedAt() time.Time {
	return room.createdAt
}

func (room *Room) UserCount() int {
	room.mutex.RLock()
	defer room.mutex.RUnlock()
	return len(room.users)
}

func (room *Room) DeleteUser(removeId string) {
	room.mutex.Lock()
	defer room.mutex.Unlock()
	removedUser, ok := room.users[removeId]
	if !ok {
		log.Println("Trying to remove user that's not in the room")
		return
	}
	for userId, user := range room.users {
		if userId != removeId {
			for _, track := range removedUser.localTracks {
				user.RemoveTrack(track)
			}
		}
	}
	err := removedUser.connection.Close()
	if err != nil {
		log.Printf("Can't close webrtc peer connection: %v\n", err)
	}
	delete(room.users, removeId)
	removedUser.onExit()
	close(removedUser.msgChan)
	go room.SendRoomStateUpdate()
}

func (room *Room) DeleteRoom() {
	room.mutex.Lock()
	defer room.mutex.Unlock()
	for _, user := range room.users {
		err := user.connection.Close()
		if err != nil {
			log.Printf("Can't close webrtc peer connection: %v\n", err)
		}
		user.onExit()
		close(user.msgChan)
	}
	room.users = map[string]*User{}
}

func (room *Room) SendRoomStateUpdate() {
	room.mutex.RLock()
	defer room.mutex.RUnlock()
	for _, user := range room.users {
		state := Message{
			UsersCount: len(room.users),
		}
		user.msgChan <- state
	}
}
