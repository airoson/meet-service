package sfu

// type WSHandler struct {
// 	roomManager RoomManager
// 	upgrader    websocket.Upgrader
// }

// func GetWSHandler(rm RoomManager) WSHandler {
// 	return WSHandler{
// 		upgrader:
// 	}
// }

// func RunSFUServer(conn *websocket.Conn, room *Room, userId string) {
// 	//Read messages from websocket and process them
// 	go func() {
// 		for {
// 			messageType, p, err := conn.ReadMessage()
// 			log.Println("Receive message from user ", userId)
// 			if err != nil {
// 				room.RemoveUser(userId)
// 				log.Println("Error while receiving message from websocket connection: ", err)
// 				return
// 			}
// 			if messageType == websocket.TextMessage {
// 				err = room.ProcessMessage(userId, p)
// 				if err != nil {
// 					log.Println("Error while processing message: ", err)
// 					return
// 				}
// 			}
// 		}
// 	}()

// 	//Read message channel and send back messages to user
// 	go func() {
// 		for {
// 			msg := room.ReceiveMessage(userId)
// 			log.Println("Sending back messages ro user ", userId)
// 			err := conn.WriteMessage(websocket.TextMessage, msg)
// 			if err != nil {
// 				log.Println("Error while sending back message to socket: ", err)
// 				return
// 			}
// 		}
// 	}()
// }
