package core

import (
	"encoding/json"
	"log"
)

func QueueEvent(pi *PrivateInfoS, evt Event, ui *UserInfo) {
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
	case EventTypeFile:
		eventBody, err = json.Marshal(&evt.Data.EventDataFile)
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
	if err != nil {
		log.Println(err)
	}
	// log.Println("QUEUED_EVENT: ", string(eventBody))
	if evt.EventType != EventTypeIntroduce {
		ret, err := pi.EncryptSign(ui.Publickey, string(eventBody))
		if err != nil {
			log.Println("Unable to EncryptSign:", err)
			return
		}
		pi.DB.Save(&QueuedEvent{
			Body:     []byte(ret),
			Endpoint: ui.Endpoint,
		})
		return
	}
	pi.DB.Save(&QueuedEvent{
		Body:     eventBody,
		Endpoint: ui.Endpoint,
	})
}

func (pi *PrivateInfoS) GetAllQueuedEvents() (qevts []*QueuedEvent) {
	pi.DB.Find(&qevts)
	return qevts
}

func (pi *PrivateInfoS) GetQueuedEvent(queuedEventID int) (qevt *QueuedEvent) {
	qevt = &QueuedEvent{}
	pi.DB.First(qevt, "id = ?", queuedEventID)
	return qevt
}
