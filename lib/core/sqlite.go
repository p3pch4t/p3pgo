package core

import (
	"log"
	"os"
	"path"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

var storePath string = ""

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func OpenSqlite(newStorePath string) {
	storePath = newStorePath
	log.Println("OpenSqlite(): opening sqlite database in:", storePath)
	os.MkdirAll(storePath, 0750)
	var err error
	DB, err = gorm.Open(sqlite.Open(path.Join(storePath, "p3p.db")), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	DB.AutoMigrate(&UserInfo{})
	DB.AutoMigrate(&PrivateInfoS{})
	DB.AutoMigrate(&QueuedEvent{})
	DB.AutoMigrate(&Message{})
	PrivateInfo.Refresh()
	go queueRunner()
	InitReachableLocal()
}
