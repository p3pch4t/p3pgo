package main

/*
#include <stdint.h>
*/
import (
	"C"
	"encoding/json"
	"log"
	"time"

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

var a []*core.PrivateInfoS

//export OpenPrivateInfo
func OpenPrivateInfo(storePath *C.char, accountName *C.char, endpointPath *C.char) int {
	pi := core.OpenPrivateInfo(C.GoString(storePath), C.GoString(accountName), C.GoString(endpointPath))
	a = append(a, pi)
	return len(a) - 1
}

//export ShowSetup
func ShowSetup(piId int) bool {
	return !a[piId].IsAccountReady()
}

//export CreateSelfInfo
func CreateSelfInfo(piId int, username *C.char, email *C.char, bitSize int) bool {
	a[piId].Create(C.GoString(username), C.GoString(email), bitSize)
	return true
}

//export GetAllUserInfo
func GetAllUserInfo(piId int) *C.char {
	ids := a[piId].GetAllUserIDs()
	b, err := json.Marshal(ids)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
}

//export AddUserByPublicKey
func AddUserByPublicKey(piId int, publickey *C.char, username *C.char, endpoint *C.char) int {
	ui, err := a[piId].CreateUserByPublicKey(C.GoString(publickey), C.GoString(username), core.Endpoint(C.GoString(endpoint)), true)
	if err != nil {
		log.Fatalln(err)
	}
	return int(ui.ID)
}

//export ForceSendIntroduceEvent
func ForceSendIntroduceEvent(piId int, uid int) bool {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	ui.SendIntroduceEvent(a[piId])
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

//export GetUserInfoMessages
func GetUserInfoMessages(piId int, UserInfoID int) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(UserInfoID))
	if err != nil {
		log.Fatalln(err)
	}
	msgs := core.GetMessagesByUserInfo(a[piId], ui)
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
func GetMessageType(piId int, msgID int) *C.char {
	//msg := core.GetMessageByID(a[piId], msgID)
	return C.CString(string(core.MessageTypeText))
}

//export GetMessageText
func GetMessageText(piId int, msgID int) *C.char {
	msg := core.GetMessageByID(a[piId], msgID)
	return C.CString(msg.Body)
}

//export GetMessageReceivedTimestamp
func GetMessageReceivedTimestamp(piId int, msgID int) int64 {
	msg := core.GetMessageByID(a[piId], msgID)
	return msg.CreatedAt.UnixMicro()
}

//export GetMessageIsIncoming
func GetMessageIsIncoming(piId int, msgID int) bool {
	msg := core.GetMessageByID(a[piId], msgID)
	return msg.Incoming
}

// ---UserInfo

//export GetUserInfoId
func GetUserInfoId(piId int, uid int) int64 {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return int64(ui.ID)
}

//export GetPrivateInfoId
func GetPrivateInfoId(piId int) int64 {
	return int64(a[piId].ID)
}

//export GetUserInfoPublicKeyArmored
func GetUserInfoPublicKeyArmored(piId int, uid int) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(ui.Publickey)
}

//export GetPrivateInfoPublicKeyArmored
func GetPrivateInfoPublicKeyArmored(piId int) *C.char {
	return C.CString(a[piId].PublicKey)
}

//export GetUserInfoUsername
func GetUserInfoUsername(piId int, uid int) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	//b, _ := json.Marshal(ui)
	return C.CString(ui.Username)
}

//export GetPrivateInfoUsername
func GetPrivateInfoUsername(piId int) *C.char {
	return C.CString(a[piId].Username)
}

//export SetUserInfoUsername
func SetUserInfoUsername(piId int, uid int, username *C.char) {
	username0 := C.GoString(username)
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	ui.Username = username0
	a[piId].DB.Save(&ui)
}

//export SetPrivateInfoUsername
func SetPrivateInfoUsername(piId int, username *C.char) {
	a[piId].Username = C.GoString(username)
	a[piId].DB.Save(a[piId])
}

//export SetPrivateInfoEepsiteDomain
func SetPrivateInfoEepsiteDomain(piId int, eepsite *C.char) {
	a[piId].Endpoint = core.Endpoint("i2p://" + string(C.GoString(eepsite)) + "/")
	a[piId].DB.Save(a[piId])
}

//export GetUserInfoEndpoint
func GetUserInfoEndpoint(piId int, uid int) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(ui.Endpoint))
}

//export SetUserInfoEndpoint
func SetUserInfoEndpoint(piId int, uid int, endpoint *C.char) {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	ui.Endpoint = core.Endpoint(string(C.GoString(endpoint)))
	a[piId].DB.Save(&ui)
}

//export GetPrivateInfoEndpoint
func GetPrivateInfoEndpoint(piId int) *C.char {
	return C.CString(string(a[piId].Endpoint))
}

//export SetPrivateInfoEndpoint
func SetPrivateInfoEndpoint(piId int, endpoint *C.char) {
	a[piId].Endpoint = core.Endpoint(string(C.GoString(endpoint)))
	a[piId].DB.Save(a[piId])
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
func SendMessage(piId int, uid int64, text *C.char) {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	core.SendMessage(a[piId], ui, core.MessageTypeText, C.GoString(text))
}

//export CreateFileStoreElement
func CreateFileStoreElement(piId int, uid uint, fileInChatPath *C.char, localFilePath *C.char) int64 {
	ui, err := a[piId].GetUserInfoByID(uid)
	if err != nil {
		return -1
	}
	fi := a[piId].CreateFileStoreElement(ui.GetKeyID(), "", C.GoString(fileInChatPath), C.GoString(localFilePath), time.Now().UnixMicro(), "")
	fi.Announce(a[piId])
	return int64(fi.ID)
}

//export GetFileStoreElementLocalPath
func GetFileStoreElementLocalPath(piId int, fseId uint) *C.char {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(fse.LocalPath())
	return C.CString(fse.LocalPath())
}

//export GetFileStoreElementIsDownloaded
func GetFileStoreElementIsDownloaded(piId int, fseId uint) bool {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.IsDownloaded()
}

//export GetFileStoreElementSizeBytes
func GetFileStoreElementSizeBytes(piId int, fseId uint) int64 {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.SizeBytes
}

//export GetFileStoreElementPath
func GetFileStoreElementPath(piId int, fseId uint) *C.char {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(fse.Path)
}

//export SetFileStoreElementPath
func SetFileStoreElementPath(piId int, fseId uint, newPath *C.char) {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	fse.Path = C.GoString(newPath)
	a[piId].DB.Save(&fse)
}

//export GetFileStoreElementIsDeleted
func GetFileStoreElementIsDeleted(piId int, fseId uint) bool {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.IsDeleted
}

//export SetFileStoreElementIsDeleted
func SetFileStoreElementIsDeleted(piId int, fseId uint, isDeleted bool) {
	fse, err := core.GetFileStoreById(a[piId], fseId)
	if err != nil {
		log.Fatalln(err)
	}
	fse.IsDeleted = isDeleted
	a[piId].DB.Save(&fse)
}

//export GetUserInfoFileStoreElements
func GetUserInfoFileStoreElements(piId int, UserInfoID int) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(UserInfoID))
	if err != nil {
		log.Fatalln(err)
	}
	msgs := core.GetFileStoreElementsByUserInfo(a[piId], ui)
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
