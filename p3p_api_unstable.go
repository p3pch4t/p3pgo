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
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
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

//export GetUserInfoMessages
func GetUserInfoMessages(UserInfoID int) *C.char {
	ui, err := core.GetUserInfoByID(uint(UserInfoID))
	if err != nil {
		log.Fatalln(err)
	}
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
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return int64(ui.ID)
}

//export GetPrivateInfoId
func GetPrivateInfoId() int64 {
	return int64(core.PrivateInfo.ID)
}

//export GetUserInfoPublicKeyArmored
func GetUserInfoPublicKeyArmored(uid int) *C.char {
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(ui.Publickey)
}

//export GetPrivateInfoPublicKeyArmored
func GetPrivateInfoPublicKeyArmored() *C.char {
	return C.CString(core.PrivateInfo.PublicKey)
}

//export GetUserInfoUsername
func GetUserInfoUsername(uid int) *C.char {
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
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
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
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
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(ui.Endpoint))
}

//export SetUserInfoEndpoint
func SetUserInfoEndpoint(uid int, endpoint *C.char) {
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
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
	ui, err := core.GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	core.SendMessage(ui, core.MessageTypeText, C.GoString(text))
}

//export CreateFileStoreElement
func CreateFileStoreElement(uid uint, fileInChatPath *C.char, localFilePath *C.char) int64 {
	ui, err := core.GetUserInfoByID(uid)
	if err != nil {
		return -1
	}
	fi := core.CreateFileStoreElement(ui.GetKeyID(), "", C.GoString(fileInChatPath), C.GoString(localFilePath), time.Now().UnixMicro())
	fi.Announce()
	return int64(fi.ID)
}

//export GetFileStoreElementLocalPath
func GetFileStoreElementLocalPath(fseId uint) *C.char {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(fse.LocalPath())
	return C.CString(fse.LocalPath())
}

//export GetFileStoreElementIsDownloaded
func GetFileStoreElementIsDownloaded(fseId uint) bool {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.IsDownloaded()
}

//export GetFileStoreElementSizeBytes
func GetFileStoreElementSizeBytes(fseId uint) int64 {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.SizeBytes
}

//export GetFileStoreElementPath
func GetFileStoreElementPath(fseId uint) *C.char {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(fse.Path)
}

//export SetFileStoreElementPath
func SetFileStoreElementPath(fseId uint, newPath *C.char) {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	fse.Path = C.GoString(newPath)
	core.DB.Save(&fse)
}

//export GetFileStoreElementIsDeleted
func GetFileStoreElementIsDeleted(fseId uint) bool {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.IsDeleted
}

//export SetFileStoreElementIsDeleted
func SetFileStoreElementIsDeleted(fseId uint, isDeleted bool) {
	fse, err := core.GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	fse.IsDeleted = isDeleted
	core.DB.Save(&fse)
}

//export GetUserInfoFileStoreElements
func GetUserInfoFileStoreElements(UserInfoID int) *C.char {
	ui, err := core.GetUserInfoByID(uint(UserInfoID))
	if err != nil {
		log.Fatalln(err)
	}
	msgs := core.GetFileStoreElementsByUserInfo(ui)
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

//FileStoreElement putFileStoreElement(
//UserInfo ui, {
//required File? localFile,
//required String fileInChatPath,
//required String? uuid,
//required bool shouldFetch,
//}) =>
//throw UnimplementedError();
