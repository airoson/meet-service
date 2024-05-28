import React, {useState, useRef, useContext} from 'react'
import pencil from '../pencil.png'
import checkmark from '../checkmark.png'
import fetchr from './Fetchr'
import { UserContext } from '../App'

function UserInfo({ field, name, init }) {
    const [update, setUpdate] = useState(init != "" ? init : "-")
    const [changing, setChanging] = useState(false)
    const {user, setUser} = useContext(UserContext)
    const text = useRef(0)

    const changeButton = (e) => {
        e.preventDefault()
        console.log("Setting read only")
        text.current.removeAttribute("readOnly")
        if(init == "") setUpdate(init)
        setChanging(true)
    }

    const patchUser = (e) => {
        e.preventDefault()
        let body = {}
        body[name] = update
        fetchr("http://localhost:8080/api/user", "PATCH", {"Content-Type": "application/json"}, 
        JSON.stringify(body), (response) => {
            if(response.status == 200) {
                alert("Successfully changed")
            }else {
                if(response.body != "") {
                    response.json().then(json => {
                        if(json.message != undefined) alert(json.message)
                    })
                }
                setUpdate(init != "" ? init : "-")
            }
        }, user, setUser)
        setChanging(false)
    }

    return (
        <div className="user-info">
            {field}: 
            <span><input type="text" onChange={e => setUpdate(e.target.value)} ref={text} value={update} readOnly className="user-info-text"/></span>
            {!changing ? 
            <button className="change-field" onClick={changeButton}><img src={pencil} className="change-field-image"/></button>
        :<button className="change-field" onClick={patchUser}><img src={checkmark} className="change-field-image"/></button>}
        </div>
    )
}

export default UserInfo
