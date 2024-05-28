import React, { useContext, useState, useEffect, useRef } from 'react'
import { useParams } from 'react-router'
import fetchr from './Fetchr'
import { UserContext } from '../App'

const config = {
  iceServers: [
    { urls: "stun:stun.l.google.com:19302" },
    {
      credential: "openrelayproject",
      username: "openrelayproject",
      urls: "turn:openrelay.metered.ca:80"
    }
  ],
  sdpSemantics: "unified-plan"
}
const pc = new RTCPeerConnection(config)
let socket = null;

const displayMediaOptions = {
  video: {
    displaySurface: "browser",
  },
  audio: {
    suppressLocalAudioPlayback: false,
  },
  preferCurrentTab: false,
  selfBrowserSurface: "exclude",
  systemAudio: "include",
  surfaceSwitching: "include",
  monitorTypeSurfaces: "include",
};
let makingOffer = false
let ignoreOffer = false

function Room() {
  const { user, setUser } = useContext(UserContext)
  const [streams, setStreams] = useState({})
  const [midToUserMap, setMidToUserMap] = useState({})
  const [shownNames, setShownNames] = useState({})
  const [remoteVideos, setRemoteVideos] = useState({})
  const [roomState, setRoomState] = useState({})
  const videoRefs = useRef([])
  const selfVideoRef = useRef(0)
  const [roomSettings, setRoomSettings] = useState({ videoSource: 1, usernameSource: 0, variants: [{ title: "Enter name", val: "" }] })
  const { roomId } = useParams()

  const handleSocketMessage = async (event) => {
    let polite = true
    let { sdp, type, candidate, sdpMLineIndex, usernameFragment, sdpMid, streamMapping, usersCount, callStart } = JSON.parse(event.data)
    try {
      if (sdp) {
        console.log("receive sdp ", type)
        const offerCollision = type === "offer" && (makingOffer || pc.signalingState !== "stable");
        ignoreOffer = !polite && offerCollision
        if (ignoreOffer) {
          return
        }
        console.log("set remote sdp")
        await pc.setRemoteDescription({
          type,
          sdp
        })
        if (type === "offer") {
          console.log("set local sdp")
          await pc.setLocalDescription()
          let description = pc.localDescription
          console.log("sending sdp answer")
          socket.send(JSON.stringify({
            sdp: description.sdp,
            type: "answer"
          }))
        }
      } else if (candidate) {
        console.log("receive ice candidate")
        try {
          await pc.addIceCandidate(new RTCIceCandidate({
            candidate,
            sdpMLineIndex,
            sdpMid,
            usernameFragment
          }))
        } catch (err) {
          if (!ignoreOffer) {
            throw err
          }
        }
      } else if (streamMapping) {
        let mapping = streamMapping.split(":")
        let mid = mapping[0]
        let userId = mapping[1]
        let shownName = mapping[2]
        let sns = { ...shownName }
        sns[userId] = shownName
        setShownNames(sns)
        let mtmap = { ...midToUserMap }
        mtmap[mid] = userId
        setMidToUserMap(mtmap)
        console.log(`receive mapping (${mid}) to ${userId} with name ${shownName}`)
      }else if (usersCount) {
        setRoomState({...roomState, usersCount})
      }else if(callStart) {
        setRoomState({...roomState, callStart: callStart})
      }
    } catch (err) {
      console.log(err)
    }
  }

  const connect = (e) => {
    e.preventDefault()
    let shownName = roomSettings.variants[roomSettings.usernameSource].val
    console.log("Start WebSocket connection...")
    socket = new WebSocket(`ws://localhost:8080/api/room/ws?roomId=${roomId}&shownName=${shownName}`)
    socket.addEventListener("open", async () => {
      console.log("Web socket connection was established")
      try {
        let stream
        if (roomSettings.videoSource == 1) {
          stream = await navigator.mediaDevices.getUserMedia({ audio: true, video: true })
        } else {
          stream = await navigator.mediaDevices.getDisplayMedia(displayMediaOptions)
        }
        setStreams({ "self": stream })
        selfVideoRef.current.srcObject = stream
        stream.getTracks().forEach(track => pc.addTrack(track))
      } catch (err) {
        console.log(err)
      }
    })
    socket.addEventListener("message", handleSocketMessage)
  }

  useEffect(() => {
    try {
      fetchr("http://localhost:8080/api/user", "GET", {}, null, (response) => {
        if (response.status === 200) {
          response.json().then(json => {
            let variants = []
            if (json.name != "") {
              variants.push({ title: json.name, val: json.name })
            }
            if (json.email != "") {
              variants.push({ title: "Email " + json.email, val: json.email })
            }
            if (json.phone != "") {
              variants.push({ title: "Tel " + json.phone, val: json.phone })
            }
            setRoomSettings({ ...roomSettings, variants: [...roomSettings.variants, ...variants] })
          })
        }
      }, user, setUser)
    } catch (err) {
      console.log("can't fetch user: ", err)
    }
  }, [])

  useEffect(() => {
    pc.ontrack = ({ track, receiver }) => {
      //let stream = streams[0]
      console.log("ontrack:")
      console.log(receiver)
      console.log(track)
      let transceiver = pc.getTransceivers().filter(t => t.receiver == receiver)[0]
      if (transceiver == undefined) {
        console.log("Can't find transceiver ")
        return
      }
      let mid = transceiver.mid
      console.log(`mid: (${mid})`)
      let userId = midToUserMap[mid]
      if (userId == undefined) {
        console.log("Can't find user id for mid=", mid)
        return
      }
      if (!(userId in streams)) {
        console.log("Creating new video, mid ", mid, " userId ", userId)
        let newStream = new MediaStream()
        newStream.addTrack(track)

        let addedVideo = <video ref={el => {
          videoRefs.current[videoRefs.length] = el
          console.log("Video is mounted")

          if (el === null) {
            console.log("video is unmounted")
          }
          console.log("new stream", newStream)
          el.srcObject = newStream
        }} autoPlay={true} controls={true} className="room-video"></video>

        let rv = { ...remoteVideos }
        rv[userId] = addedVideo
        setRemoteVideos(rv)

        let ns = { ...streams }
        ns[userId] = newStream
        setStreams(ns)
      } else {
        console.log("Getting existing stream, mid=", mid)
        let stream = streams[userId]
        stream.addTrack(track)
      }
    }

    pc.onnegotiationneeded = async () => {
      try {
        makingOffer = true
        await pc.setLocalDescription()
        let description = pc.localDescription
        console.log("sending sdp ", description.type)
        socket.send(JSON.stringify({
          type: description.type,
          sdp: description.sdp
        }))
      } catch (err) {
        console.log(err)
      } finally {
        makingOffer = false
      }
    }

    pc.onicecandidate = ({ candidate }) => {
      if (candidate == null) return
      console.log("sending ice candidate")
      try {
        socket.send(JSON.stringify({
          candidate: candidate.candidate,
          sdpMid: candidate.sdpMid,
          sdpMLineId: candidate.sdpMLineId,
          usernameFragment: candidate.usernameFragment
        }))
      } catch (err) {
        console.log(err)
      }
    }
  }, [midToUserMap, remoteVideos, streams])

  const showRemoteVideos = () => {
    if (remoteVideos.length == 0) return <div className="remote-video-name">No connected users</div>
    for (let userId of Object.keys(remoteVideos)) {
      let name = shownNames[userId]
      return (
        <div className="remote-video-holder">
          {<div className="remote-video-name">{name != undefined ? name : "Unknown user"}</div>}
          {remoteVideos[userId]}
        </div>)
    }
  }

  return (
    <div>
      <div className="room-interface-holder">
        <div className="self-video-holder">
          <h3>Your video:</h3>
          <video id="self-video" autoPlay={true} controls={true} className="room-video" ref={selfVideoRef}></video>
        </div>
        <div className="room-controls">
          <h3>
            Controls:
          </h3>
          <div className="controls-panel">
            <div className="controls-name">View:</div>
            <div className="controls">
              <label>
                Camera <input type="radio" name="source" checked={roomSettings.videoSource === 1} onChange={() => setRoomSettings({ ...roomSettings, videoSource: 1 })} />
              </label>
              <label>
                Screen <input type="radio" name="source" checked={roomSettings.videoSource === 2} onChange={() => setRoomSettings({ ...roomSettings, videoSource: 2 })} />
              </label>
            </div>
            <div className="controls-name">Shown name:</div>
            <div className="controls">
              <select onChange={e => { setRoomSettings({ ...roomSettings, usernameSource: e.target.value }); console.log("Option changed: ", e.target.value) }}>
                {roomSettings.variants.map((variant, i) => <option value={i} key={i} checked={i == 0}>{variant.title}</option>)}
              </select>
              <input type="text" id="shown-name" readOnly={roomSettings.usernameSource != 0} value={roomSettings.variants[roomSettings.usernameSource].val}
                onChange={e => {
                  roomSettings.variants[roomSettings.usernameSource] = { ...roomSettings.variants[roomSettings.usernameSource], val: e.target.value }
                  setRoomSettings({ ...roomSettings, variants: roomSettings.variants })
                }} />
            </div>
            <button type="button" className="connect-button" onClick={connect} >Connect</button>
          </div>
          <div>
            {roomState.callStart != undefined ? <div>Call start: {roomState.callStart}</div>: ""}
            {roomState.usersCount != undefined ? <div>Users count: {roomState.usersCount}</div>: ""}
          </div>
        </div>
      </div>
      
      {showRemoteVideos()}
    </div>
  )
}

export default Room
