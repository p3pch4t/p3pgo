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

func GetMessageByID(msgID int) Message {
	var msg Message
	DB.First(&msg, "id = ?", msgID)
	return msg
}

func GetMessagesByUserInfo(ui UserInfo) []Message {
	var msgs []Message
	DB.Where("key_id = ?", ui.GetKeyID()).Order("created_at DESC").Find(&msgs)
	return msgs
}

func GetFileStoreElementsByUserInfo(ui UserInfo) []FileStoreElement {
	var fselms []FileStoreElement
	DB.Where("internal_key_id = ?", ui.GetKeyID()).Order("created_at DESC").Find(&fselms)
	return fselms
}

func SendMessage(ui UserInfo, messageType MessageType, text string) {
	log.Println("SendMessage", ui.GetKeyID(), messageType)
	DB.Save(&Message{KeyID: ui.GetKeyID(), Incoming: false, Body: text})
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
	QueueEvent(evt, ui)
}
