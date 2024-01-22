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

func main() {}

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
func OpenPrivateInfo(storePath *C.char, accountName *C.char, endpointPath *C.char, isMini bool) int {
	pi := core.OpenPrivateInfo(C.GoString(storePath), C.GoString(accountName), C.GoString(endpointPath), isMini)
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
	msgs := a[piId].GetMessagesByUserInfo(ui)
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
	msg := a[piId].GetMessageByID(msgID)
	return C.CString(msg.Body)
}

//export GetMessageReceivedTimestamp
func GetMessageReceivedTimestamp(piId int, msgID int) int64 {
	msg := a[piId].GetMessageByID(msgID)
	return msg.CreatedAt.UnixMicro()
}

//export GetMessageIsIncoming
func GetMessageIsIncoming(piId int, msgID int) bool {
	msg := a[piId].GetMessageByID(msgID)
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
	a[piId].SendMessage(ui, core.MessageTypeText, C.GoString(text))
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
	fse, err := a[piId].GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	log.Println(fse.LocalPath())
	return C.CString(fse.LocalPath())
}

//export GetFileStoreElementIsDownloaded
func GetFileStoreElementIsDownloaded(piId int, fseId uint) bool {
	fse, err := a[piId].GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.IsDownloaded()
}

//export GetFileStoreElementSizeBytes
func GetFileStoreElementSizeBytes(piId int, fseId uint) int64 {
	fse, err := a[piId].GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.SizeBytes
}

//export GetFileStoreElementPath
func GetFileStoreElementPath(piId int, fseId uint) *C.char {
	fse, err := a[piId].GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(fse.Path)
}

//export SetFileStoreElementPath
func SetFileStoreElementPath(piId int, fseId uint, newPath *C.char) {
	fse, err := a[piId].GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	fse.Path = C.GoString(newPath)
	a[piId].DB.Save(&fse)
}

//export GetFileStoreElementIsDeleted
func GetFileStoreElementIsDeleted(piId int, fseId uint) bool {
	fse, err := a[piId].GetFileStoreById(fseId)
	if err != nil {
		log.Fatalln(err)
	}
	return fse.IsDeleted
}

//export SetFileStoreElementIsDeleted
func SetFileStoreElementIsDeleted(piId int, fseId uint, isDeleted bool) {
	fse, err := a[piId].GetFileStoreById(fseId)
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
	msgs := a[piId].GetFileStoreElementsByUserInfo(ui)
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

//export GetUserInfoEndpointStats
func GetUserInfoEndpointStats(piId int, uid int64) uint {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return ui.GetEndpointStats(a[piId]).ID
}

// --------- QueuedEvents

//export GetQueuedEventIDs
func GetQueuedEventIDs(piId int) *C.char {
	qevts := a[piId].GetAllQueuedEvents()
	var qevtsId []uint
	for i := range qevts {
		qevtsId = append(qevtsId, qevts[i].ID)
	}
	b, err := json.Marshal(qevtsId)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
}

//export GetQueuedEventCreatedAt
func GetQueuedEventCreatedAt(piId int, queuedEventId int) int64 {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return qevt.CreatedAt.UnixMicro()
}

//export GetQueuedEventDeletedAt
func GetQueuedEventDeletedAt(piId int, queuedEventId int) int64 {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	if qevt.DeletedAt.Valid {
		return qevt.DeletedAt.Time.UnixMicro()
	}
	return 0
}

//export GetQueuedEventUpdatedAt
func GetQueuedEventUpdatedAt(piId int, queuedEventId int) int64 {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return qevt.UpdatedAt.UnixMicro()
}

//export GetQueuedEventLastRelayed
func GetQueuedEventLastRelayed(piId int, queuedEventId int) int64 {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return qevt.LastRelayed.UnixMicro()
}

//export GetQueuedEventBody
func GetQueuedEventBody(piId int, queuedEventId int) []byte {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return qevt.Body
}

//export GetQueuedEventEndpoint
func GetQueuedEventEndpoint(piId int, queuedEventId int) *C.char {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return C.CString(string(qevt.Endpoint))
}

//export GetQueuedEventRelayTries
func GetQueuedEventRelayTries(piId int, queuedEventId int) int {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return qevt.RelayTries
}

//export GetQueuedEventEndpointStats
func GetQueuedEventEndpointStats(piId int, queuedEventId int) uint {
	qevt := a[piId].GetQueuedEvent(queuedEventId)
	return qevt.GetEndpointStats(a[piId]).ID
}

//export GetEndpointStatsCreatedAt
func GetEndpointStatsCreatedAt(piId int, endpointStatsId int) int64 {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return estats.CreatedAt.UnixMicro()
}

//export GetEndpointStatsUpdatedAt
func GetEndpointStatsUpdatedAt(piId int, endpointStatsId int) int64 {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return estats.UpdatedAt.UnixMicro()
}

//export GetEndpointStatsDeletedAt
func GetEndpointStatsDeletedAt(piId int, endpointStatsId int) int64 {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	if estats.DeletedAt.Valid {
		return estats.DeletedAt.Time.UnixMicro()
	}
	return 0
}

//export GetEndpointStatsEndpoint
func GetEndpointStatsEndpoint(piId int, endpointStatsId int) *C.char {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return C.CString(estats.Endpoint)
}

//export GetEndpointStatsLastContactOut
func GetEndpointStatsLastContactOut(piId int, endpointStatsId int) int64 {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return estats.LastContactOut.UnixMicro()
}

//export GetEndpointStatsLastContactIn
func GetEndpointStatsLastContactIn(piId int, endpointStatsId int) int64 {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return estats.LastContactIn.UnixMicro()
}

//export GetEndpointStatsFailInRow
func GetEndpointStatsFailInRow(piId int, endpointStatsId int) int {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return estats.FailInRow
}

//export GetEndpointStatsCurrentDelay
func GetEndpointStatsCurrentDelay(piId int, endpointStatsId int) int {
	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
	return estats.CurrentDelay
}
