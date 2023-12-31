package core

import (
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
	Text    string      `json:"text,omitempty"`
	MsgUUID string      `json:"msguuid,omitempty"`
	Type    MessageType `json:"type,omitempty"`
}

type EventDataFile struct {
	Uuid       string `json:"file_uuid,omitempty"`
	HttpPath   string `json:"http_path,omitempty"`
	Path       string `json:"path,omitempty"` // the in chat path, eg /Apps/Calendar.xdc
	Sha512sum  string `json:"sha512sum,omitempty"`
	SizeBytes  int64  `json:"sizeBytes,omitempty"`
	IsDeleted  bool   `json:"isDeleted,omitempty"`
	ModifyTime int64  `json:"modifyTime,omitempty"`
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
	case EventTypeFile:
		evt.tryProcessFile(pi)
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

// EventTypeFile             EventType = "file"
func (evt *Event) tryProcessFile(pi *PrivateInfoS) {
	log.Println("evt.tryProcessFile")
	if evt.InternalKeyID == "" {
		log.Println("warn! unknown evt.InternalKeyID")
		evt.InternalKeyID = "___UNKNOWN___"
		return
	}
	fse := pi.CreateFileStoreElement(
		StringToKeyID(evt.InternalKeyID),
		evt.Data.EventDataFile.Uuid,
		evt.Data.EventDataFile.Path,
		"",
		evt.Data.EventDataFile.ModifyTime,
		evt.Data.EventDataFile.HttpPath,
	)
	ui, err := pi.GetUserInfoByKeyID(evt.InternalKeyID)
	if err != nil {
		log.Println(err)
		return
	}
	for i := range pi.FileStoreElementCallback {
		pi.FileStoreElementCallback[i](pi, ui, &fse, false)
	}
}
