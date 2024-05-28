package sfu

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"

	"github.com/pion/webrtc/v4"
)

var config = webrtc.Configuration{
	ICEServers: []webrtc.ICEServer{
		{
			URLs: []string{"stun:stun.l.google.com:19302"},
		},
		{
			URLs:           []string{"turn:openrelay.metered.ca:80"},
			Username:       "openrelayproject",
			Credential:     "openrelayproject",
			CredentialType: webrtc.ICECredentialTypePassword,
		},
	},
	SDPSemantics: webrtc.SDPSemanticsUnifiedPlan,
}

type User struct {
	id             string
	connection     *webrtc.PeerConnection
	msgChan        chan Message
	mutex          *sync.RWMutex
	midMap         map[string]string
	localTracks    []webrtc.TrackLocal
	addTrackToRoom func(string, *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error)
	onExit         func()
}

type UserInterface interface {
	GetId() string
	ReceiveOffer(offer string) (*webrtc.SessionDescription, error)
	ReceiveAnswer(answer string) error
	AddTrack(track webrtc.TrackLocal) *webrtc.RTPSender
	RemoveTrack(track webrtc.TrackLocal)
	AddIceCandidate(candidate webrtc.ICECandidateInit)
}

func (user *User) GetId() string {
	return user.id
}

func (user *User) AddTrack(track webrtc.TrackLocal, from string) {
	log.Printf("[User.AddTrack] Adding track with id %s to user with id %s\n", user.id, track.ID())
	user.mutex.Lock()
	defer user.mutex.Unlock()
	rtpTransceiver, err := user.connection.AddTransceiverFromTrack(track)
	if err != nil {
		log.Println("Can't add local track: ", err)
	}
	rtpTransceiver.SetMid(strconv.Itoa(rand.Int() % 10000))
	user.midMap[track.ID()] = from
	log.Printf("Add new track from %s with mid: %s\n", from, rtpTransceiver.Mid())
	msg := Message{
		StreamMapping: rtpTransceiver.Mid() + ":" + from,
	}
	user.msgChan <- msg
}

func (user *User) RemoveTrack(track webrtc.TrackLocal) {
	user.mutex.Lock()
	defer user.mutex.Unlock()
	for _, sender := range user.connection.GetSenders() {
		if sender.Track().ID() == track.ID() {
			user.connection.RemoveTrack(sender)
		}
	}
}

func (user *User) AddIceCandidate(candidate webrtc.ICECandidateInit) {
	err := user.connection.AddICECandidate(candidate)
	if err != nil {
		log.Println("Can't add ICE candidate: ", err)
	}
}

func (user *User) OnIceCandidate(candidate *webrtc.ICECandidate) {
	if candidate == nil {
		return
	}
	jsonCandidate := candidate.ToJSON()
	outMsg := Message{
		Candidate:     jsonCandidate.Candidate,
		SdpMid:        *jsonCandidate.SDPMid,
		SdpMLineIndex: *jsonCandidate.SDPMLineIndex,
	}
	user.msgChan <- outMsg
}

func (user *User) OnNegotiationRequest() {
	log.Println("Renegotiation...")
	offer, err := user.connection.CreateOffer(nil)
	if err != nil {
		log.Println("Can't create offer: ", err)
		return
	}
	user.connection.SetLocalDescription(offer)
	description := user.connection.LocalDescription()

	if description != nil {
		outMsg := Message{
			Sdp:         description.SDP,
			MessageType: "offer",
		}
		log.Println("Sending sdp offer")
		user.msgChan <- outMsg
	}
}

func (user *User) onTrack(track *webrtc.TrackRemote, _ *webrtc.RTPReceiver) {
	log.Println("New track for user ", user.id)
	trackLocal, err := user.addTrackToRoom(user.id, track)
	user.localTracks = append(user.localTracks, trackLocal)
	if err != nil {
		log.Println("Can't add remove track: ", err)
		return
	}
	for {
		rtp, _, err := track.ReadRTP()
		if err != nil {
			log.Println("Can't read from remote track: ", err)
			return
		}
		err = trackLocal.WriteRTP(rtp)
		if err != nil {
			log.Println("Can' write to local track: ", err)
			return
		}
	}
}

func CreateUser(userId string, addTrackToRoom func(string, *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error), onExit func()) (*User, error) {
	user := User{}
	peer, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("Can't create peer connection: ", err)
		return nil, fmt.Errorf("can't create user: %v", err)
	}
	user.id = userId
	user.mutex = &sync.RWMutex{}
	user.msgChan = make(chan Message, 5)
	user.addTrackToRoom = addTrackToRoom
	user.localTracks = []webrtc.TrackLocal{}
	user.midMap = map[string]string{}
	user.onExit = onExit
	// user.rtpSenders = map[string]*webrtc.RTPSender{}

	peer.OnICECandidate(user.OnIceCandidate)
	peer.OnNegotiationNeeded(user.OnNegotiationRequest)
	peer.OnTrack(user.onTrack)
	user.connection = peer
	return &user, nil
}

func (user *User) ReceiveOffer(offer string) (*webrtc.SessionDescription, error) {
	sdp := webrtc.SessionDescription{
		SDP:  offer,
		Type: webrtc.SDPTypeOffer,
	}
	err := user.connection.SetRemoteDescription(sdp)
	if err != nil {
		log.Println()
		return nil, fmt.Errorf("can't add remote offer: %v", err)
	}

	answer, err := user.connection.CreateAnswer(nil)
	if err != nil {
		log.Println("Can't create answer: ", err)
		return nil, fmt.Errorf("can't create answer: %v", err)
	}
	err = user.connection.SetLocalDescription(answer)
	if err != nil {
		log.Println("Can't set local description: ", err)
		return nil, fmt.Errorf("can't set local description: %v", err)
	}
	return &answer, nil
}

func (user *User) ReceiveAnswer(answer string) error {
	sdp := webrtc.SessionDescription{
		SDP:  answer,
		Type: webrtc.SDPTypeAnswer,
	}
	err := user.connection.SetRemoteDescription(sdp)
	if err != nil {
		log.Println("Can't set remote answer: ", err)
		return fmt.Errorf("can't set remote answer: %v", err)
	}
	return nil
}
