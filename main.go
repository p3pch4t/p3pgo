package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/events"
	reachable_local "git.mrcyjanek.net/p3pch4t/p3pgo/lib/reachable/local"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

func main() {
	reachable_local.InitReachableLocal()
	evt := events.Event{
		EventType: events.EventTypeIntroduceRequest,
		Data: events.EventDataMixed{EventDataIntroduce: events.EventDataIntroduce{
			PublicKey: core.PrivateInfo.PublicKey,
			Endpoint:  "local://127.0.0.1:3000",
			Username:  core.PrivateInfo.Username,
		}},
	}
	evt.RandomizeUuid()
	b, err := json.Marshal(evt)
	if err != nil {
		log.Fatalln(err)
	}
	time.Sleep(time.Millisecond * 250)
	resp, err := core.PrivateInfo.EncryptSign(core.SelfUser.Publickey, string(b))
	if err != nil {
		log.Fatalln("Failed to encryptsign", err)
	}
	b = []byte(resp)
	log.Println("Sending:", len(b))

	http.Post("http://127.0.0.1:3000/", "application/json", bytes.NewReader(b))
	time.Sleep(time.Second * 3)
}
