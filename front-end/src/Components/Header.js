import React, { useContext, useState } from 'react'
import { UserContext } from '../App'

function Header() {
  const {user, setUser} = useContext(UserContext)
  const logout = async (e) => {
    e.preventDefault()
    await fetch("http://localhost:8080/api/auth/logout", {
      method: "POST",
      credentials: "include"
    }).then(response => {
      if(response.status == 200) {
        alert("Logout successfully")
        window.localStorage.removeItem("accessToken")
        window.localStorage.removeItem("identifier")
        setUser({accessToken: null, identifier: null})
        window.location.href = "/"
      }else {
        alert("Can't log out :/")
      }
    })
  }
  const getButtonHolder = () => {
    if (user.accessToken == null) {
      return (
        <div className="header-buttons-holder">
          <button className="header-button" onClick={() => window.location.href = "./login"}>Login</button>
          <button className="header-button" onClick={() => window.location.href = "./signup"}>Signup</button>
        </div>
      )
    } else {
      return (
        <div className="header-buttons-holder">
          <span>{user.identifier}</span>
          <button className="header-button" onClick={() => window.location.href = "./user"}>User profile</button>
          <button className="header-button" onClick={logout}>Logout</button>
        </div>
      )
    }
  }
  return (
    <header>
      <a className="home-url" href="/">Meet service</a>
      {getButtonHolder()}
    </header>
  )
}

export default Header
