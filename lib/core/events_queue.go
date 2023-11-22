package core

import (
	"encoding/json"
	"log"
)

func QueueEvent(evt Event, ui UserInfo) {
	if evt.Uuid == "" {
		evt.RandomizeUuid()
	}
	var eventBody []byte
	var err error
	switch evt.EventType {
	case EventTypeIntroduce:
		eventBody, err = json.Marshal(&evt.Data.EventDataIntroduce)
	case EventTypeIntroduceRequest:
		eventBody, err = json.Marshal(&evt.Data.EventDataIntroduceRequest)
	case EventTypeMessage:
		eventBody, err = json.Marshal(&evt.Data.EventDataMessage)
	case EventTypeFileRequest:
		eventBody, err = json.Marshal(&evt.Data.EventDataFileRequest)
	case EventTypeFile:
		eventBody, err = json.Marshal(&evt.Data.EventDataFile)
	case EventTypeFileMetadata:
		eventBody, err = json.Marshal(&evt.Data.EventDataFileMetadata)
	default:
		log.Println("WARN: Unable to queue event:", evt.EventType)
	}
	if err != nil {
		log.Println("WARN: Unable to json.Marshal event, reason:", err)
	}
	if len(eventBody) == 0 {
		log.Println("WARN: We are about to queue 0 sized eventBody")
	}
	var evtBodyDecoded interface{}
	err = json.Unmarshal(eventBody, &evtBodyDecoded)
	if err != nil {
		log.Println("WARN: Unable to json.Unmarshal event, reason:", err)
	}
	finalEvt := EventEncodable{
		EventType: evt.EventType,
		Data:      evtBodyDecoded,
		Uuid:      evt.Uuid,
	}
	eventBody, err = json.Marshal(&finalEvt)
	log.Println("final eventBody:", eventBody)
	// log.Println("QUEUED_EVENT: ", string(eventBody))
	ret, err := PrivateInfo.EncryptSign(ui.Publickey, string(eventBody))
	if err != nil {
		log.Println("Unable to EncryptSign:", err)
		return
	}
	DB.Save(&QueuedEvent{
		Body:     []byte(ret),
		Endpoint: ui.Endpoint,
	})
}
