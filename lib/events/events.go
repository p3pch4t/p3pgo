package events

import (
	"log"
	"strings"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
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
	EventType EventType      `json:"type"`
	Data      EventDataMixed `json:"data"`
	Uuid      string         `json:"uuid"`
}

func (evt *Event) RandomizeUuid() {
	evt.Uuid = "aaaaa-aaa-aaaa-aaa-aaa"
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
	EventDataIntroduceRequest
	//PublicKey string   `json:"publickey"`
	//Endpoints []string `json:"endpoints"`
	Username string `json:"username"`
}

type EventDataIntroduceRequest struct {
	PublicKey string        `json:"publickey"`
	Endpoint  core.Endpoint `json:"endpoint"`
}
type EventDataMessage struct {
	Text MessageType `json:"text"`
	Type MessageType `json:"type"`
}

type EventDataFileRequest struct {
	Uuid  string `json:"uuid"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}

type EventDataFile struct {
	Uuid  string `json:"file_uuid"`
	Start int    `json:"file_start"`
	End   int    `json:"file_end"`
	// according to golang docs
	// Array and slice values encode as JSON arrays, except that []byte encodes
	// as a base64-encoded string, and a nil slice encodes as the null JSON object.
	// https://pkg.go.dev/encoding/json#Marshal
	// So this should work just fine with p3p.dart
	// TODO: Check if it actually does.
	Bytes []byte `json:"file_bytes"`
}

type EventDataFileMetadata struct {
	Files []FileStoreElement `json:"files"`
}
type FileStoreElement struct {
	Uuid       string `json:"uuid"`
	Path       string `json:"path"`
	Sha512sum  string `json:"sha512sum"`
	SizeBytes  int    `json:"sizeBytes"`
	IsDeleted  bool   `json:"isDeleted"`
	ModifyTime int    `json:"modifyTime"`
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
	publicKey, err := crypto.NewKeyFromArmored(evt.Data.EventDataIntroduceRequest.PublicKey)
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
	core.DB.Save(ui)
	queueEvent(Event{
		EventType: EventTypeIntroduce,
		Data: EventDataMixed{
			EventDataIntroduce: EventDataIntroduce{
				EventDataIntroduceRequest: EventDataIntroduceRequest{
					PublicKey: core.SelfUser.Publickey,
					Endpoint:  core.SelfUser.Endpoint,
				},
				Username: core.SelfUser.Username,
			},
		},
	},
		ui.Endpoint)
}

// EventTypeMessage          EventType = "message"
// EventTypeFileRequest      EventType = "file.request"
// EventTypeFile             EventType = "file"
// EventTypeFileMetadata     EventType = "file.metadata"
