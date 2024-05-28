import React, { useState } from 'react'

function Signup() {
  const [cred, setCred] = useState({ identifier: "", password: "" })

  const submitForm = (e) => {
    e.preventDefault()
    fetch("http://localhost:8080/api/auth/signup", {
      method: "POST",
      headers: {
        "Content-Type": "application/json"
      },
      body: JSON.stringify({
        identifier: cred.identifier,
        password: cred.password
      })
    }).then(response => {
      response.json().then(json => console.log(json))
      if(response.status == 200) alert("User is created")
      else alert(`Can't create user`)
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
        <button type="submit" onClick={submitForm}>Sign up</button>
      </form>
    </div>
  )
}

export default Signup
