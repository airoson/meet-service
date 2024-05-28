import React, { useState, useContext, useEffect } from 'react'
import loading from '../loading.gif'
import fetchr from './Fetchr.js'
import { UserContext } from '../App'
import UserInfo from './UserInfo'
import { Link } from 'react-router-dom'
import cross from '../cross.png'

function User() {
    const [userInfo, setUserInfo] = useState(null)
    const [roomInfo, setRoomInfo] = useState([])
    const [room, setRoom] = useState({ startsAt: "", title: "", description: "" })
    const { user, setUser } = useContext(UserContext)
    useEffect(() => {
        fetchr("http://localhost:8080/api/user", "GET", {},
            null, (response) => {
                if (response.status != 200) {
                    alert("Can't load user")
                } else {
                    response.json().then(json => {
                        let userId = json.userId
                        setUserInfo({
                            name: json.name,
                            email: json.email,
                            phone: json.phone
                        });
                        fetchr(`http://localhost:8080/api/room?userId=${userId}`, "GET", {}, null, response => {
                            if (response.status == 200) {
                                response.json().then(json => {
                                    setRoomInfo(json.data)
                                })
                            } else {
                                alert("Can't read room info")
                            }
                        }, user, setUser)
                    })
                }
            }, user, setUser)
    }, [])

    const createRoom = (e) => {
        e.preventDefault()
        let body = {
            title: room.title,
            description: room.description
        }
        if (room.startsAt != "") {
            body.startsAt = room.startsAt
        }
        console.log(room.startsAt)
        fetchr("http://localhost:8080/api/room", "POST", { "Content-Type": "application/json" }, JSON.stringify(body),
            response => {
                if (response.status == 200) {
                    response.json().then(json => {
                        setRoomInfo([...roomInfo, {
                            title: room.title,
                            description: room.description,
                            startsAt: room.startsAt,
                            createdAt: json.createdAt,
                            usersCount: json.usersCount,
                            active: json.active,
                            id: json.id
                        }])
                    })
                } else {
                    if (response.body != "") {
                        response.json().then(json => {
                            alert("Can't create room: ", json.message)
                        })
                    } else alert("Can't create room")
                }
            }, user, setUser)
    }

    const deleteRoom = (e, roomId) => {
        e.preventDefault()
        fetchr(`http://localhost:8080/api/room?roomId=${roomId}`, "DELETE", {}, null, response => {
            if (response.status == 200) {
                alert("Room was deleted")
                setRoomInfo([...roomInfo.filter(ri => ri.id != roomId)])
            }else {
                alert("Can't delete room")
            }
        }, user, setUser)
    }

    return (
        <div className="user-holder">
            {userInfo == null || roomInfo == null ? (
                <div>
                    <img src={loading} className="loading-icon" />
                </div>) : (
                <div className="user-info-holder">
                    <h2>Current user</h2>
                    <UserInfo field="Name" name={"name"} init={userInfo.name} />
                    <UserInfo field="Email" name={"email"} init={userInfo.email} />
                    <UserInfo field="Phone" name={"phone"} init={userInfo.email} />
                    <h2>Rooms</h2>
                    {roomInfo.map((info, i) =>
                    (
                        <div className="room-holder">
                            <h3>{info.title}</h3> <button onClick={e => deleteRoom(e, info.id)} className="room-delete-button"><img src={cross} /></button>
                            <ul>
                                <li>Created at: {info.createdAt}</li>
                                <li>Starts at: {info.startsAt != undefined ? info.startsAt : "-"}</li>
                                <li>Description: {info.description}</li>
                                <li>Users count: {info.usersCount}</li>
                                <li>Active: {String(info.active)}</li>
                                <li><Link to={`/room/${info.id}`}>Link to room</Link></li>
                            </ul>
                        </div>
                    )
                    )}
                    <h2>Create room</h2>
                    <form className="create-room">
                        <div>
                            <label className="create-room-label">
                                Title <input type="text" value={room.title} onChange={e => setRoom({ ...room, title: e.target.value })} />
                            </label>
                            <label className="create-room-label">
                                Starts at <input type="datetime-local" value={room.startsAt} onChange={e => setRoom({ ...room, startsAt: e.target.value })} />
                            </label>
                        </div>
                        <label className="create-room-label">
                            Description: <br /><textarea className="description-text" rows="5" cols="50" value={room.description} onChange={e => setRoom({ ...room, description: e.target.value })} />
                        </label>
                        <div style={{ display: "flex", justifyContent: "center" }}><button type="submit" onClick={createRoom} className="room-create-button">Submit</button></div>
                    </form>
                </div>
            )}
        </div>
    )
}

export default User
