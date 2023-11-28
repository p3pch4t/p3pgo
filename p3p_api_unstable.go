package main

/*
#include <stdint.h>
*/
import (
	"C"
	"encoding/json"
	"log"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
)

//export Print
func Print(s *C.char) {
	log.Printf("p3p.Print(): %s", C.GoString(s))
}

//export HealthCheck
func HealthCheck() bool {
	log.Println("HealthCheck: ok")
	return true
}

//export InitStore
func InitStore(storePath *C.char) bool {
	core.OpenSqlite(C.GoString(storePath))
	return true
}

//export ShowSetup
func ShowSetup() bool {
	core.PrivateInfo.Refresh()
	log.Println("ShowSetup:", len(core.PrivateInfo.Passphrase))
	return len(core.PrivateInfo.Passphrase) == 0
}

//export CreateSelfInfo
func CreateSelfInfo(username *C.char, email *C.char, bitSize int) bool {
	core.PrivateInfo.Create(C.GoString(username), C.GoString(email), bitSize)
	return true
}

//export GetAllUserInfo
func GetAllUserInfo() *C.char {
	ids := core.GetAllUserIDs()
	b, err := json.Marshal(ids)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
}

//export AddUserByPublicKey
func AddUserByPublicKey(publickey *C.char, username *C.char, endpoint *C.char) int {
	ui, err := core.CreateUserByPublicKey(C.GoString(publickey), C.GoString(username), core.Endpoint(C.GoString(endpoint)), true)
	if err != nil {
		log.Fatalln(err)
	}
	return int(ui.ID)
}

//export ForceSendIntroduceEvent
func ForceSendIntroduceEvent(uid int) bool {
	ui := core.GetUserInfoByID(uint(uid))
	ui.SendIntroduceEvent()
	return true
}

//export GetUserDetailsByURL
func GetUserDetailsByURL(url *C.char) *C.char {
	endpointStr := core.Endpoint(C.GoString(url))
	dui, err := core.DiscoverUserByURL(endpointStr.GetHost())
	if err != nil {
		log.Println(err)
		return C.CString("{}")
	}
	b, err := json.Marshal(dui)
	if err != nil {
		log.Println(err)
		return C.CString("{}")
	}
	return C.CString(string(b))
}

//export GetChatMessages
func GetChatMessages(UserInfoID int) *C.char {
	var ui core.UserInfo = core.GetUserInfoByID(uint(UserInfoID))
	msgs := core.GetMessagesByUserInfo(ui)
	var msgids []uint
	for i := range msgs {
		msgids = append(msgids, msgs[i].ID)
	}
	b, err := json.Marshal(msgids)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
}

// ---Message

//export GetMessageType
func GetMessageType(msgID int) *C.char {
	//msg := core.GetMessageByID(msgID)
	return C.CString(string(core.MessageTypeText))
}

//export GetMessageText
func GetMessageText(msgID int) *C.char {
	msg := core.GetMessageByID(msgID)
	return C.CString(msg.Body)
}

//export GetMessageReceivedTimestamp
func GetMessageReceivedTimestamp(msgID int) int64 {
	msg := core.GetMessageByID(msgID)
	return msg.CreatedAt.UnixMicro()
}

//export GetMessageIsIncoming
func GetMessageIsIncoming(msgID int) bool {
	msg := core.GetMessageByID(msgID)
	return msg.Incoming
}

// ---UserInfo

//export GetUserInfoId
func GetUserInfoId(uid int) int64 {
	ui := core.GetUserInfoByID(uint(uid))
	return int64(ui.ID)
}

//export GetPrivateInfoId
func GetPrivateInfoId() int64 {
	return int64(core.PrivateInfo.ID)
}

//export GetUserInfoPublicKeyArmored
func GetUserInfoPublicKeyArmored(uid int) *C.char {
	ui := core.GetUserInfoByID(uint(uid))
	return C.CString(ui.Publickey)
}

//export GetPrivateInfoPublicKeyArmored
func GetPrivateInfoPublicKeyArmored() *C.char {
	return C.CString(core.PrivateInfo.PublicKey)
}

//export GetUserInfoUsername
func GetUserInfoUsername(uid int) *C.char {
	ui := core.GetUserInfoByID(uint(uid))
	//b, _ := json.Marshal(ui)
	return C.CString(ui.Username)
}

//export GetPrivateInfoUsername
func GetPrivateInfoUsername() *C.char {
	return C.CString(core.PrivateInfo.Username)
}

//export SetUserInfoUsername
func SetUserInfoUsername(uid int, username *C.char) {
	username0 := C.GoString(username)
	ui := core.GetUserInfoByID(uint(uid))
	ui.Username = username0
	core.DB.Save(&ui)
}

//export SetPrivateInfoUsername
func SetPrivateInfoUsername(username *C.char) {
	core.PrivateInfo.Username = C.GoString(username)
	core.DB.Save(&core.PrivateInfo)
}

//export SetPrivateInfoEepsiteDomain
func SetPrivateInfoEepsiteDomain(eepsite *C.char) {
	core.PrivateInfo.Endpoint = core.Endpoint("i2p://" + string(C.GoString(eepsite)) + "/")
	core.DB.Save(&core.PrivateInfo)
}

//export GetUserInfoEndpoint
func GetUserInfoEndpoint(uid int) *C.char {
	ui := core.GetUserInfoByID(uint(uid))
	return C.CString(string(ui.Endpoint))
}

//export SetUserInfoEndpoint
func SetUserInfoEndpoint(uid int, endpoint *C.char) {
	ui := core.GetUserInfoByID(uint(uid))
	ui.Endpoint = core.Endpoint(string(C.GoString(endpoint)))
	core.DB.Save(&ui)
}

//export GetPrivateInfoEndpoint
func GetPrivateInfoEndpoint() *C.char {
	return C.CString(string(core.PrivateInfo.Endpoint))
}

//export SetPrivateInfoEndpoint
func SetPrivateInfoEndpoint(endpoint *C.char) {
	core.PrivateInfo.Endpoint = core.Endpoint(string(C.GoString(endpoint)))
	core.DB.Save(&core.PrivateInfo)
}

// ---PublicKey

//export GetPublicKeyFingerprint
func GetPublicKeyFingerprint(armored *C.char) *C.char {
	str := C.GoString(armored)
	publicKey, err := crypto.NewKeyFromArmored(str)
	if err != nil {
		log.Println("WARN: Unable to armor public key: ", err, "armored:", str)
		return C.CString("")
	}
	return C.CString(publicKey.GetFingerprint())
}

//export SendMessage
func SendMessage(uid int64, text *C.char) {
	core.SendMessage(core.GetUserInfoByID(uint(uid)), core.MessageTypeText, C.GoString(text))
}
