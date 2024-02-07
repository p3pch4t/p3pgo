package core

import (
	"github.com/google/uuid"
	"log"

	"gorm.io/gorm"
)

type Message struct {
	gorm.Model
	KeyID    string
	Body     string
	Incoming bool
}

func (pi *PrivateInfoS) GetMessageByID(msgID int) Message {
	var msg Message
	pi.DB.First(&msg, "id = ?", msgID)
	return msg
}

func (pi *PrivateInfoS) GetMessagesByUserInfo(ui *UserInfo) []Message {
	var msgs []Message
	pi.DB.Where("key_id = ?", ui.GetKeyID()).Order("created_at DESC").Find(&msgs)
	return msgs
}

func (pi *PrivateInfoS) SendMessage(ui *UserInfo, messageType MessageType, text string) {
	log.Println("SendMessage", ui.GetKeyID(), messageType)
	pi.DB.Save(&Message{KeyID: ui.GetKeyID(), Incoming: false, Body: text})
	evt := Event{
		InternalKeyID: ui.GetKeyID(),
		EventType:     EventTypeMessage,
		Data: EventDataMixed{
			EventDataMessage: EventDataMessage{
				Text:    text,
				Type:    messageType,
				MsgUUID: uuid.NewString(),
			},
		},
		Uuid: "",
	}
	QueueEvent(pi, evt, ui)
}
