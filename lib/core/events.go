package core

import (
	"gorm.io/gorm"
	"log"
	"strings"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/google/uuid"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

type EventType string

const (
	EventTypeUnimplemented    EventType = "unimplemented"
	EventTypeIntroduce        EventType = "introduce"
	EventTypeIntroduceRequest EventType = "introduce.request"
	EventTypeMessage          EventType = "message"
)

type Event struct {
	InternalKeyID string         `json:"-"`
	EventType     EventType      `json:"type"`
	Data          EventDataMixed `json:"data"`
	Uuid          string         `json:"uuid"`
}

type EventEncodable struct {
	EventType EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Uuid      string      `json:"uuid"`
}

func (evt *Event) RandomizeUuid() {
	evt.Uuid = uuid.New().String()
}

type EventDataMixed struct {
	EventDataIntroduce
	EventDataIntroduceRequest
	EventDataMessage
}

type MessageType string

const (
	MessageTypeUnimplemented MessageType = "unimplemented"
	MessageTypeText          MessageType = "text"
	MessageTypeService       MessageType = "service"
	MessageTypeHidden        MessageType = "hidden"
)

type EventDataIntroduce struct {
	PublicKey     string                          `json:"publickey,omitempty"`
	Endpoint      Endpoint                        `json:"endpoints,omitempty"`
	Username      string                          `json:"username,omitempty"`
	FilesMetadata map[string]*SharedFilesMetadata `json:"filesMetadata,omitempty"`
}

type SharedFilesMetadata struct {
	gorm.Model     `json:"-"`
	DBKeyID        string   `json:"-"`
	KeyPart        string   `json:"keyPart,omitempty"`
	FilesEndpoint  Endpoint `json:"filesEndpoint,omitempty"`
	Authentication string   `json:"authentication,omitempty"`
}

type EventDataIntroduceRequest struct {
	SelfPublicKey string   `json:"selfpublickey,omitempty"`
	Endpoint      Endpoint `json:"endpoint,omitempty"`
}
type EventDataMessage struct {
	Text    string      `json:"text,omitempty"`
	MsgUUID string      `json:"msguuid,omitempty"`
	Type    MessageType `json:"type,omitempty"`
}

func (evt *Event) TryProcess(pi *PrivateInfoS) {
	for i := range pi.EventCallback {
		pi.EventCallback[i](pi, evt)
	}
	switch evt.EventType {
	case EventTypeIntroduce:
		evt.tryProcessIntroduce(pi)
	case EventTypeIntroduceRequest:
		evt.tryProcessIntroduceRequest(pi)
	case EventTypeMessage:
		evt.tryProcessMessage(pi)
	default:
		log.Println("WARN: Unhandled event, type:", evt.EventType)
	}
}

// EventTypeUnimplemented    EventType = "unimplemented"
// EventTypeIntroduce        EventType = "introduce"
func (evt *Event) tryProcessIntroduce(pi *PrivateInfoS) {
	log.Println("evt.tryProcessIntroduce")
	if evt.EventType != EventTypeIntroduce {
		log.Fatalln("invalid type.")
	}
	ui, err := pi.CreateUserByPublicKey(
		evt.Data.EventDataIntroduce.PublicKey,
		evt.Data.EventDataIntroduce.Username,
		evt.Data.EventDataIntroduce.Endpoint,
		false,
	)
	log.Println("new introduction:", evt.Data.EventDataIntroduce.Username, ui.Username, err)

	fs := evt.Data.EventDataIntroduce.FilesMetadata
	for i := range fs {
		pi.DB.Delete(&SharedFilesMetadata{}, "files_endpoint = ? AND key_part = ?", fs[i].FilesEndpoint, i)
		pi.DB.Save(&SharedFilesMetadata{
			DBKeyID:        ui.GetKeyID(),
			KeyPart:        i,
			FilesEndpoint:  fs[i].FilesEndpoint,
			Authentication: fs[i].Authentication,
		})
	}

	for i := range pi.IntroduceCallback {
		pi.IntroduceCallback[i](pi, ui, evt)
	}
}

// EventTypeIntroduceRequest EventType = "introduce.request"
func (evt *Event) tryProcessIntroduceRequest(pi *PrivateInfoS) {
	log.Println("evt.tryProcessIntroduceRequest")
	if evt.EventType != EventTypeIntroduceRequest {
		log.Fatalln("invalid type.")
	}
	publicKey, err := crypto.NewKeyFromArmored(evt.Data.PublicKey)
	if err != nil {
		log.Println("WARN: Unable to armor public key, returning.", err)
		return
	}
	var ui *UserInfo
	pi.DB.Where("fingerprint = ?", publicKey.GetFingerprint()).First(ui)
	b, err := publicKey.GetArmoredPublicKeyWithCustomHeaders("p3pgo", "")
	if err != nil {
		log.Println("WARN: Unable to publickey.GetPublicKey()")
		return
	}
	ui.Publickey = b
	ui.Fingerprint = strings.ToLower(publicKey.GetFingerprint())
	ui.Endpoint = evt.Data.EventDataIntroduceRequest.Endpoint
	pi.DB.Save(&ui)
	QueueEvent(pi, Event{
		EventType: EventTypeIntroduce,
		Data: EventDataMixed{
			EventDataIntroduce: EventDataIntroduce{
				PublicKey: pi.PublicKey,
				Endpoint:  pi.Endpoint,
				Username:  pi.Username,
			},
		},
	},
		ui)
}

// EventTypeMessage          EventType = "message"
func (evt *Event) tryProcessMessage(pi *PrivateInfoS) {
	log.Println("evt.tryProcessMessage")
	if evt.InternalKeyID == "" {
		log.Println("warn! unknown evt.InternalKeyID")
		evt.InternalKeyID = "___UNKNOWN___"
		return
	}
	if len(evt.InternalKeyID) > 16 {
		evt.InternalKeyID = evt.InternalKeyID[len(evt.InternalKeyID)-16:]
	}
	log.Println("InternalKeyID:", evt.InternalKeyID)
	msg := &Message{
		KeyID:    evt.InternalKeyID,
		Body:     string(evt.Data.EventDataMessage.Text),
		Incoming: true,
	}
	if pi.IsMini {
		log.Println("Not saving Message{}, because IsMini == true. Call `pi.DB.Save(msg)' on your own if you wish.")
	} else {
		pi.DB.Save(msg)

	}
	ui, err := pi.GetUserInfoByKeyID(evt.InternalKeyID)
	if err != nil {
		log.Println(err)
		return
	}
	for i := range pi.MessageCallback {
		pi.MessageCallback[i](pi, ui, evt, msg)
	}
}
