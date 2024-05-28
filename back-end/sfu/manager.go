package sfu

import (
	"sync"

	"github.com/google/uuid"
)

type InternalRoomManager struct {
	rooms map[string]*Room
	mutex *sync.RWMutex
}

type RoomManager interface {
	CreateRoomWithId(string)
	CreateRoom() string
	DeleteRoom(string)
	GetRoom(string) *Room
}

func GetInternalRoomManager() *InternalRoomManager {
	return &InternalRoomManager{
		rooms: map[string]*Room{},
		mutex: &sync.RWMutex{},
	}
}

func (manager *InternalRoomManager) CreateRoom() string {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	room := CreateRoom(uuid.New().String())
	manager.rooms[room.id] = room
	return room.id
}

func (manager *InternalRoomManager) CreateRoomWithId(id string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	room := CreateRoom(id)
	manager.rooms[id] = room
}

func (manager *InternalRoomManager) GetRoom(id string) *Room {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()
	return manager.rooms[id]
}

func (manager *InternalRoomManager) DeleteRoom(id string) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	delete(manager.rooms, id)
}
