package core

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// var DB *gorm.DB

var storePath string = ""
var logPath string = ""

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func GetMD5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}
func OpenPrivateInfo(newStorePath string, accountName string, endpointPath string) *PrivateInfoS {
	storePath = path.Join(newStorePath, GetMD5Hash(accountName))
	_ = os.MkdirAll(storePath, 0750)
	logPath = path.Join(storePath, "log.txt")
	logFile, err := os.Create(logPath)
	if err != nil {
		log.Fatalln(err)
	}
	mw := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(mw)
	log.Println("OpenSqlite(): logger setup!")
	log.Println("OpenSqlite(): opening sqlite database in:", storePath)
	var pi = PrivateInfoS{AccountName: accountName}
	pi.DB, err = gorm.Open(sqlite.Open(path.Join(storePath, "p3p.db")), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("DB.AutoMigrate.UserInfo", pi.DB.AutoMigrate(&UserInfo{}))
	log.Println("DB.AutoMigrate.QueuedEvent", pi.DB.AutoMigrate(&QueuedEvent{}))
	log.Println("DB.AutoMigrate.Message", pi.DB.AutoMigrate(&Message{}))
	log.Println("DB.AutoMigrate.FileStoreElement", pi.DB.AutoMigrate(&FileStoreElement{}))
	log.Println("DB.AutoMigrate.PrivateInfoS", pi.DB.AutoMigrate(&PrivateInfoS{}))
	pi.Refresh()
	go queueRunner(&pi)
	go fileStoreElementQueueRunner(&pi)
	ensureProperUserInfo(&pi)
	StartLocalServer()
	pi.InitReachableLocal(endpointPath)
	go pi.FileStoreElementDownloadLoop()
	return &pi
}

func ensureProperUserInfo(pi *PrivateInfoS) {
	var uis []UserInfo
	pi.DB.Find(&uis)
	for i := range uis {
		if uis[i].KeyID != uis[i].GetKeyID() {
			log.Println("Fixing userInfo", uis[i].KeyID, "to", uis[i].GetKeyID())
			uis[i].KeyID = uis[i].GetKeyID()
			pi.DB.Save(&uis[i])
		}
	}
}
