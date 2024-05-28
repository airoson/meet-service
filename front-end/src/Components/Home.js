import React, { useState } from 'react'

function Home() {
  const [roomId, setRoomId] = useState("")

  const connect = (e) => {
    e.preventDefault()
    window.location.href = `/room/${roomId}`
  }

  return (
    <div className="home-holder">
      <h2>Welcome!</h2>
      <div>
        <p>
          Meet service v 0.0.1
        </p>
        <p>
          Create account to start new room, or connect to existing one with id:
          <div className="home-form-holder">
            <label>Id <input type="text" value={roomId} onChange={e => setRoomId(e.target.value)} /></label>
            <button id="home-connect" onClick={connect}>Connect</button>
          </div>
        </p>
      </div>
    </div>
  )
}

export default Home
