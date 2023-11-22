package core

import (
	"gorm.io/gorm"
	"log"
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
	DB.Where("key_id = ?", ui.KeyID).Order("created_at DESC").Find(&msgs)
	return msgs
}

func SendMessage(ui UserInfo, messageType MessageType, text string) {
	log.Println("SendMessage", ui.KeyID, messageType)
	DB.Save(&Message{KeyID: ui.KeyID, Incoming: false, Body: text})
	evt := Event{
		InternalKeyID: ui.KeyID,
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
