package events

import (
	"log"
	"strings"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
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
	EventTypeFileRequest      EventType = "file.request"
	EventTypeFile             EventType = "file"
	EventTypeFileMetadata     EventType = "file.metadata"
)

type Event struct {
	InternalKeyID string         `json:"-"`
	EventType     EventType      `json:"type"`
	Data          EventDataMixed `json:"data"`
	Uuid          string         `json:"uuid"`
}

func (evt *Event) RandomizeUuid() {
	evt.Uuid = uuid.New().String()
}

type EventDataMixed struct {
	EventDataIntroduce
	EventDataIntroduceRequest
	EventDataMessage
	EventDataFileRequest
	EventDataFile
	EventDataFileMetadata
}

type MessageType string

const (
	MessageTypeUnimplemented MessageType = "unimplemented"
	MessageTypeText          MessageType = "text"
	MessageTypeService       MessageType = "service"
	MessageTypeHidden        MessageType = "hidden"
)

type EventDataIntroduce struct {
	// EventDataIntroduceRequest
	PublicKey string        `json:"publickey"`
	Endpoint  core.Endpoint `json:"endpoints"`
	Username  string        `json:"username,omitempty"`
}

type EventDataIntroduceRequest struct {
	SelfPublicKey string        `json:"selfpublickey,omitempty"`
	Endpoint      core.Endpoint `json:"endpoint,omitempty"`
}
type EventDataMessage struct {
	Text MessageType `json:"text,omitempty"`
	Type MessageType `json:"type,omitempty"`
}

type EventDataFileRequest struct {
	Uuid  string `json:"uuid,omitempty"`
	Start int    `json:"start,omitempty"`
	End   int    `json:"end,omitempty"`
}

type EventDataFile struct {
	Uuid  string `json:"file_uuid,omitempty"`
	Start int    `json:"file_start,omitempty"`
	End   int    `json:"file_end,omitempty"`
	// according to golang docs
	// Array and slice values encode as JSON arrays, except that []byte encodes
	// as a base64-encoded string, and a nil slice encodes as the null JSON object.
	// https://pkg.go.dev/encoding/json#Marshal
	// So this should work just fine with p3p.dart
	// TODO: Check if it actually does.
	Bytes []byte `json:"file_bytes,omitempty"`
}

type EventDataFileMetadata struct {
	Files []FileStoreElement `json:"files,omitempty"`
}
type FileStoreElement struct {
	Uuid       string `json:"uuid,omitempty"`
	Path       string `json:"path,omitempty"`
	Sha512sum  string `json:"sha512sum,omitempty"`
	SizeBytes  int    `json:"sizeBytes,omitempty"`
	IsDeleted  bool   `json:"isDeleted,omitempty"`
	ModifyTime int    `json:"modifyTime,omitempty"`
}

func (evt *Event) TryProcess() {
	switch evt.EventType {
	case EventTypeIntroduce:
		evt.tryProcessIntroduce()
	case EventTypeIntroduceRequest:
		evt.tryProcessIntroduceRequest()
	default:
		log.Println("WARN: Unhandled event, type:", evt.EventType)
	}
}

// EventTypeUnimplemented    EventType = "unimplemented"
// EventTypeIntroduce        EventType = "introduce"
func (evt *Event) tryProcessIntroduce() {
	if evt.EventType != EventTypeIntroduce {
		log.Fatalln("invalid type.")
	}
	publicKey, err := crypto.NewKeyFromArmored(evt.Data.EventDataIntroduce.PublicKey)
	if err != nil {
		log.Println("WARN: Unable to armor public key, returning.")
		return
	}
	var ui core.UserInfo
	core.DB.Where("fingerprint = ?", publicKey.GetFingerprint()).First(&ui)
	b, err := publicKey.GetArmoredPublicKeyWithCustomHeaders("p3pgo", "")
	if err != nil {
		log.Println("WARN: Unable to publickey.GetPublicKey()")
		return
	}
	ui.Publickey = b
	ui.Fingerprint = strings.ToLower(publicKey.GetFingerprint())
	ui.Username = evt.Data.EventDataIntroduce.Username
	ui.Endpoint = evt.Data.EventDataIntroduce.Endpoint
	core.DB.Save(ui)
	log.Panicln("new introduction:", evt.Data.EventDataIntroduce.Username)
}

// EventTypeIntroduceRequest EventType = "introduce.request"
func (evt *Event) tryProcessIntroduceRequest() {
	if evt.EventType != EventTypeIntroduceRequest {
		log.Fatalln("invalid type.")
	}
	publicKey, err := crypto.NewKeyFromArmored(evt.Data.PublicKey)
	if err != nil {
		log.Println("WARN: Unable to armor public key, returning.", err)
		return
	}
	var ui core.UserInfo
	core.DB.Where("fingerprint = ?", publicKey.GetFingerprint()).First(&ui)
	b, err := publicKey.GetArmoredPublicKeyWithCustomHeaders("p3pgo", "")
	if err != nil {
		log.Println("WARN: Unable to publickey.GetPublicKey()")
		return
	}
	ui.Publickey = b
	ui.Fingerprint = strings.ToLower(publicKey.GetFingerprint())
	ui.Endpoint = evt.Data.EventDataIntroduceRequest.Endpoint
	core.DB.Save(&ui)
	queueEvent(Event{
		EventType: EventTypeIntroduce,
		Data: EventDataMixed{
			EventDataIntroduce: EventDataIntroduce{
				PublicKey: core.SelfUser.Publickey,
				Endpoint:  core.SelfUser.Endpoint,
				Username:  core.SelfUser.Username,
			},
		},
	},
		ui.Endpoint)
}

// EventTypeMessage          EventType = "message"
func (evt *Event) tryProcessMessage() {

}

// EventTypeFileRequest      EventType = "file.request"
// EventTypeFile             EventType = "file"
// EventTypeFileMetadata     EventType = "file.metadata"
