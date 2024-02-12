package main

/*
#include <stdint.h>
*/
import (
	"C"
	"encoding/json"
	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"log"
)

/*
 * This is unstable api used in p3pch4t. Other projects are recommended to use Go library directly
 * If you also need to use the C api, then this api is the way to go.
 * I'll do my best to keep it stable but some breaking changes may happen.
 */

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

//export GetUserInfoEndpointStats
func GetUserInfoEndpointStats(piId int, uid int64) uint {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	return ui.GetEndpointStats(a[piId]).ID
}

//export GetUserInfoSharedFilesMetadataIDs
func GetUserInfoSharedFilesMetadataIDs(piId int, uid int64) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	list := ui.GetReceivedSharedFilesMetadataIDs(a[piId])
	b, err := json.Marshal(list)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
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

////export GetEndpointStatsLastContactIn
//func GetEndpointStatsLastContactIn(piId int, endpointStatsId int) int64 {
//	estats := a[piId].GetEndpointStatsByID(endpointStatsId)
//	return estats.LastContactIn.UnixMicro()
//}

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

//export CreateFile
func CreateFile(piId int, uid int64, localFilePath *C.char, remoteFilePath *C.char) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		return C.CString(err.Error())
	}
	err = a[piId].CreateFile(ui, C.GoString(localFilePath), C.GoString(remoteFilePath))
	if err != nil {
		return C.CString(err.Error())
	}
	return C.CString("")
}

//export GetSharedFilesIDs
func GetSharedFilesIDs(piId int, uid int64) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	ids := a[piId].GetSharedFilesIDs(ui)
	b, err := json.Marshal(ids)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
}

// func (pi *PrivateInfoS) GetSharedFileById(id uint) (sf *SharedFile) {

//export GetSharedFileID
func GetSharedFileID(piId int, fileId uint) uint {
	f := a[piId].GetSharedFileById(fileId)
	return f.ID
}

//export GetSharedFileCreatedAt
func GetSharedFileCreatedAt(piId int, fileId uint) int64 {
	f := a[piId].GetSharedFileById(fileId)
	return f.CreatedAt.UnixMicro()
}

//export GetSharedFileUpdatedAt
func GetSharedFileUpdatedAt(piId int, fileId uint) int64 {
	f := a[piId].GetSharedFileById(fileId)
	return f.UpdatedAt.UnixMicro()
}

//export GetSharedFileDeletedAt
func GetSharedFileDeletedAt(piId int, fileId uint) int64 {
	f := a[piId].GetSharedFileById(fileId)
	if !f.DeletedAt.Valid {
		return 0
	}
	return f.DeletedAt.Time.UnixMicro()
}

//export GetSharedFileSharedFor
func GetSharedFileSharedFor(piId int, fileId uint) *C.char {
	f := a[piId].GetSharedFileById(fileId)
	return C.CString(f.SharedFor)
}

//export GetSharedFileSha512Sum
func GetSharedFileSha512Sum(piId int, fileId uint) *C.char {
	f := a[piId].GetSharedFileById(fileId)
	return C.CString(f.Sha512Sum)
}

//export GetSharedFileLastEdit
func GetSharedFileLastEdit(piId int, fileId uint) int64 {
	f := a[piId].GetSharedFileById(fileId)
	return f.LastEdit.UnixMicro()
}

//export GetSharedFileFilePath
func GetSharedFileFilePath(piId int, fileId uint) *C.char {
	f := a[piId].GetSharedFileById(fileId)
	return C.CString(f.FilePath)
}

//export GetSharedFileLocalFilePath
func GetSharedFileLocalFilePath(piId int, fileId uint) *C.char {
	f := a[piId].GetSharedFileById(fileId)
	return C.CString(f.LocalFilePath)
}

//export GetSharedFileSizeBytes
func GetSharedFileSizeBytes(piId int, fileId uint) int64 {
	f := a[piId].GetSharedFileById(fileId)
	return f.SizeBytes
}

//export DeleteSharedFile
func DeleteSharedFile(piId int, fileId uint) {
	f := a[piId].GetSharedFileById(fileId)
	a[piId].DeleteSharedFile(f)
}

//export GetUserInfoSharedFilesIDs
func GetUserInfoSharedFilesIDs(piId int, uid int64) *C.char {
	ui, err := a[piId].GetUserInfoByID(uint(uid))
	if err != nil {
		log.Fatalln(err)
	}
	list := ui.GetReceivedSharedFilesMetadataIDs(a[piId])
	b, err := json.Marshal(list)
	if err != nil {
		log.Fatalln(err)
	}
	return C.CString(string(b))
}

//export GetReceivedSharedFilesMetadataID
func GetReceivedSharedFilesMetadataID(piId int, uid uint) uint {
	sfm := a[piId].GetReceivedSharedFile(uid)
	return sfm.ID
}

// KeyPart        string   `json:"keyPart,omitempty"`
// FilesEndpoint  Endpoint `json:"filesEndpoint,omitempty"`
// Authentication string   `json:"authentication,omitempty"`
//

//export GetReceivedSharedFilesMetadataDBKeyID
func GetReceivedSharedFilesMetadataDBKeyID(piId int, uid uint) *C.char {
	sfm := a[piId].GetReceivedSharedFile(uid)
	return C.CString(sfm.DBKeyID)
}

//export GetReceivedSharedFilesMetadataKeyPart
func GetReceivedSharedFilesMetadataKeyPart(piId int, uid uint) *C.char {
	sfm := a[piId].GetReceivedSharedFile(uid)
	return C.CString(sfm.KeyPart)
}

//export GetReceivedSharedFilesMetadataFilesEndpoint
func GetReceivedSharedFilesMetadataFilesEndpoint(piId int, uid uint) *C.char {
	sfm := a[piId].GetReceivedSharedFile(uid)
	return C.CString(string(sfm.FilesEndpoint))
}

//export GetReceivedSharedFilesMetadataAuthentication
func GetReceivedSharedFilesMetadataAuthentication(piId int, uid uint) *C.char {
	sfm := a[piId].GetReceivedSharedFile(uid)
	return C.CString(sfm.Authentication)
}
