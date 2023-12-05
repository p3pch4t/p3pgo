package main

import (
	"encoding/json"
	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
	"github.com/google/uuid"
	"log"
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
			Text:    "Message Text",
			Type:    core.MessageTypeText,
			MsgUUID: uuid.NewString(),
		}},
	}
	evt.RandomizeUuid()
	b, err := json.Marshal(evt)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(string(b))
	return
}
