package core

import (
	"io"
	"log"
	"os"
	"path"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

var storePath string = ""
var logPath string = ""

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func OpenSqlite(newStorePath string) {
	storePath = newStorePath

	logPath = path.Join(storePath, "log.txt")
	logFile, err := os.Create(logPath)
	if err != nil {
		log.Fatalln(err)
	}
	mw := io.MultiWriter(logFile, os.Stderr)
	log.SetOutput(mw)
	log.Println("OpenSqlite(): logger setup!")
	log.Println("OpenSqlite(): opening sqlite database in:", storePath)
	os.MkdirAll(storePath, 0750)
	DB, err = gorm.Open(sqlite.Open(path.Join(storePath, "p3p.db")), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("DB.AutoMigrate.UserInfo", DB.AutoMigrate(&UserInfo{}))
	log.Println("DB.AutoMigrate.QueuedEvent", DB.AutoMigrate(&QueuedEvent{}))
	log.Println("DB.AutoMigrate.Message", DB.AutoMigrate(&Message{}))
	log.Println("DB.AutoMigrate.FileStoreElement", DB.AutoMigrate(&FileStoreElement{}))
	PrivateInfo.Refresh()
	go queueRunner()
	InitReachableLocal()
}
