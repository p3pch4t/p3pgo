package core

import (
	"log"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	KeyID    string
	Body     string
	Incoming bool
}

func GetMessageByID(pi *PrivateInfoS, msgID int) Message {
	var msg Message
	pi.DB.First(&msg, "id = ?", msgID)
	return msg
}

func GetMessagesByUserInfo(pi *PrivateInfoS, ui UserInfo) []Message {
	var msgs []Message
	pi.DB.Where("key_id = ?", ui.GetKeyID()).Order("created_at DESC").Find(&msgs)
	return msgs
}

func GetFileStoreElementsByUserInfo(pi *PrivateInfoS, ui UserInfo) []FileStoreElement {
	var fselms []FileStoreElement
	pi.DB.Where("internal_key_id = ?", ui.GetKeyID()).Order("created_at DESC").Find(&fselms)
	return fselms
}

func SendMessage(pi *PrivateInfoS, ui UserInfo, messageType MessageType, text string) {
	log.Println("SendMessage", ui.GetKeyID(), messageType)
	pi.DB.Save(&Message{KeyID: ui.GetKeyID(), Incoming: false, Body: text})
	evt := Event{
		InternalKeyID: ui.GetKeyID(),
		EventType:     EventTypeMessage,
		Data: EventDataMixed{
			EventDataMessage: EventDataMessage{
				Text: text,
				Type: messageType,
			},
		},
		Uuid: "",
	}
	QueueEvent(pi, evt, ui)
}
