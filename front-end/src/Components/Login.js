import React, { useContext, useState } from 'react'
import { UserContext } from '../App'

function Login() {
  const [cred, setCred] = useState({ identifier: "", password: "" })
  const {user, setUser} = useContext(UserContext)

  const submitForm = (e) => {
    e.preventDefault()
    fetch("http://localhost:8080/api/auth/login", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        identifier: cred.identifier,
        password: cred.password
      }),
      credentials: "include"
    }).then(response => {
      if (response.status == 200) {
        response.json().then(json => {
          let accessToken = json["accessToken"]
          window.localStorage.setItem("accessToken", accessToken)
          setUser({accessToken: accessToken, identifier: cred.identifier})
          alert("Login completed")
          window.location.href = "/user"
        })
      }
      else if (response.body != "") {
        response.json().then(json => {
          let message = json["message"]
          if(message != undefined){
            alert(`Authentication failed: ${message}`)
          }
        })
      }
    }).catch(e => console.log(e))
  }

  return (
    <div className="form-holder">
      <form className="signup-form">
        <label>
          Email or phone: <input type="text" value={cred.identifier} onChange={(e) => setCred({ ...cred, identifier: e.target.value })} />
        </label>
        <label>
          Password: <input type="text" value={cred.password} onChange={(e) => setCred({ ...cred, password: e.target.value })} />
        </label>
        <button type="submit" onClick={submitForm}>Login</button>
      </form>
    </div>
  )
}

export default Login
