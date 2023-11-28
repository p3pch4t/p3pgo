package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func init() {
	log.Println("Loading p3pgo")
}

func main() {
	evt := core.Event{
		EventType: core.EventTypeMessage,
		Data: core.EventDataMixed{EventDataMessage: core.EventDataMessage{
			Text: "Message Text",
			Type: core.MessageTypeText,
		}},
	}
	evt.RandomizeUuid()
	b, err := json.Marshal(evt)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(b))
	return
	time.Sleep(time.Millisecond * 250)
	resp, err := core.PrivateInfo.EncryptSign(core.PrivateInfo.PublicKey, string(b))
	if err != nil {
		log.Fatalln("Failed to encryptsign", err)
	}
	b = []byte(resp)
	log.Println("Sending:", len(b))
	http.Post("http://127.0.0.1:3000/", "application/json", bytes.NewReader(b))
	time.Sleep(time.Second * 3)
}
