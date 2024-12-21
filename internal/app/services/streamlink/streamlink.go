package streamlink

import (
	"fmt"
)

type Streamlink struct {
	Twitch *TwitchAPI
}

func New() *Streamlink {
	clientId := "kimne78kx3ncx6brgo4mv6wki5h1ko"
	deviceId, err := RandomToken(32, "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789")
	if err != nil {
		fmt.Println("Error generate deviceId:", err)
	}

	return &Streamlink{
		Twitch: NewTwitch(clientId, deviceId),
	}
}
