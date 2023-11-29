package core

import (
	"log"
	"os"
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
	EventTypeFile             EventType = "file"
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
	EventDataFile
}

type MessageType string

const (
	MessageTypeUnimplemented MessageType = "unimplemented"
	MessageTypeText          MessageType = "text"
	MessageTypeService       MessageType = "service"
	MessageTypeHidden        MessageType = "hidden"
)

type EventDataIntroduce struct {
	PublicKey string   `json:"publickey,omitempty"`
	Endpoint  Endpoint `json:"endpoints,omitempty"`
	Username  string   `json:"username,omitempty"`
}

type EventDataIntroduceRequest struct {
	SelfPublicKey string   `json:"selfpublickey,omitempty"`
	Endpoint      Endpoint `json:"endpoint,omitempty"`
}
type EventDataMessage struct {
	Text string      `json:"text,omitempty"`
	Type MessageType `json:"type,omitempty"`
}

type EventDataFile struct {
	Uuid string `json:"file_uuid,omitempty"`
	// according to golang docs
	// Array and slice values encode as JSON arrays, except that []byte encodes
	// as a base64-encoded string, and a nil slice encodes as the null JSON object.
	// https://pkg.go.dev/encoding/json#Marshal
	// So this should work just fine with p3p.dart
	// TODO: Check if it actually does.
	Bytes []byte `json:"file_bytes,omitempty"`
	//Path - is the in chat path, eg /Apps/Calendar.xdc
	Path       string `json:"path,omitempty"`
	Sha512sum  string `json:"sha512sum,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
	IsDeleted  bool   `json:"isDeleted,omitempty"`
	ModifyTime int64  `json:"modifyTime,omitempty"`
}

func (evt *Event) TryProcess() {
	switch evt.EventType {
	case EventTypeIntroduce:
		evt.tryProcessIntroduce()
	case EventTypeIntroduceRequest:
		evt.tryProcessIntroduceRequest()
	case EventTypeMessage:
		evt.tryProcessMessage()
	case EventTypeFile:
		evt.tryProcessFile()
	default:
		log.Println("WARN: Unhandled event, type:", evt.EventType)
	}
}

// EventTypeUnimplemented    EventType = "unimplemented"
// EventTypeIntroduce        EventType = "introduce"
func (evt *Event) tryProcessIntroduce() {
	log.Println("evt.tryProcessIntroduce")
	if evt.EventType != EventTypeIntroduce {
		log.Fatalln("invalid type.")
	}
	ui, err := CreateUserByPublicKey(
		evt.Data.EventDataIntroduce.PublicKey,
		evt.Data.EventDataIntroduce.Username,
		evt.Data.EventDataIntroduce.Endpoint,
		false,
	)
	log.Println("new introduction:", evt.Data.EventDataIntroduce.Username, ui.Username, err)
}

// EventTypeIntroduceRequest EventType = "introduce.request"
func (evt *Event) tryProcessIntroduceRequest() {
	log.Println("evt.tryProcessIntroduceRequest")
	if evt.EventType != EventTypeIntroduceRequest {
		log.Fatalln("invalid type.")
	}
	publicKey, err := crypto.NewKeyFromArmored(evt.Data.PublicKey)
	if err != nil {
		log.Println("WARN: Unable to armor public key, returning.", err)
		return
	}
	var ui UserInfo
	DB.Where("fingerprint = ?", publicKey.GetFingerprint()).First(&ui)
	b, err := publicKey.GetArmoredPublicKeyWithCustomHeaders("p3pgo", "")
	if err != nil {
		log.Println("WARN: Unable to publickey.GetPublicKey()")
		return
	}
	ui.Publickey = b
	ui.Fingerprint = strings.ToLower(publicKey.GetFingerprint())
	ui.Endpoint = evt.Data.EventDataIntroduceRequest.Endpoint
	DB.Save(&ui)
	QueueEvent(Event{
		EventType: EventTypeIntroduce,
		Data: EventDataMixed{
			EventDataIntroduce: EventDataIntroduce{
				PublicKey: PrivateInfo.PublicKey,
				Endpoint:  PrivateInfo.Endpoint,
				Username:  PrivateInfo.Username,
			},
		},
	},
		ui)
}

// EventTypeMessage          EventType = "message"
func (evt *Event) tryProcessMessage() {
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
	DB.Save(&Message{
		KeyID:    evt.InternalKeyID,
		Body:     string(evt.Data.EventDataMessage.Text),
		Incoming: true,
	})
}

// EventTypeFile             EventType = "file"
func (evt *Event) tryProcessFile() {
	log.Println("evt.tryProcessFile")
	if evt.InternalKeyID == "" {
		log.Println("warn! unknown evt.InternalKeyID")
		evt.InternalKeyID = "___UNKNOWN___"
		return
	}
	f, err := os.CreateTemp(storePath, "tmp")
	if err != nil {
		log.Fatalln(err)
	}
	f.Write(evt.Data.EventDataFile.Bytes)
	defer f.Close()
	CreateFileStoreElement(
		StringToKeyID(evt.InternalKeyID),
		evt.Data.EventDataFile.Uuid,
		evt.Data.EventDataFile.Path,
		f.Name(),
		evt.Data.EventDataFile.ModifyTime,
	)
	//	fse.UpdateContent(false)
}
