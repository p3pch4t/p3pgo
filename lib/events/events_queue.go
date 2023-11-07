package events

import (
	"encoding/json"
	"log"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
)

func queueEvent(evt Event, endpoint core.Endpoint) {
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
	log.Println("QUEUED_EVENT: ", string(eventBody))
	core.DB.Save(&core.QueuedEvent{
		Body:     eventBody,
		Endpoint: endpoint,
	})
}
