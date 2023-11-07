package core

import (
	"log"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB
var SelfUser = UserInfo{ID: 1}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func init() {
	log.Println("init(): opening sqlite database")
	var err error
	DB, err = gorm.Open(sqlite.Open("p3p.db"), &gorm.Config{})
	if err != nil {
		log.Fatalln(err)
	}
	DB.AutoMigrate(UserInfo{})
	DB.AutoMigrate(PrivateInfoS{})
	DB.AutoMigrate(QueuedEvent{})
	PrivateInfo.Refresh()
	prepareSelfUser()
}

func prepareSelfUser() {
	DB.Where(&SelfUser).FirstOrCreate(&SelfUser)
	if SelfUser.Username == "" {
		SelfUser.Username = "SelfUser [p3pgo]"
	}
	if SelfUser.Publickey == "" {
		privateKeyObj, err := crypto.NewKeyFromArmored(PrivateInfo.PrivateKey)
		if err != nil {
			log.Fatalln(err)
		}
		SelfUser.Publickey, err = privateKeyObj.GetArmoredPublicKeyWithCustomHeaders("p3pgo", "")
		if err != nil {
			log.Fatalln(err)
		}
	}
	DB.Save(&SelfUser)
}
