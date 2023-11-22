package core

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"gorm.io/gorm"
)

type UserInfo struct {
	gorm.Model
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	Publickey   string   `json:"publickey"`
	Fingerprint string   `json:"-"`
	KeyID       string   `json:"-"`
	Endpoint    Endpoint `json:"endpoint"`
}

func GetUserInfoByID(id uint) UserInfo {
	var ui UserInfo
	DB.Find(&ui, "id = ?", id)
	return ui
}

func GetAllUserIDs() (UserInfoIDs []uint) {
	var uis []UserInfo
	DB.Find(&uis)
	for i := range uis {
		if uis[i].Fingerprint == "" && uis[i].Publickey == "" {
			DB.Delete(&uis[i])
			continue
		}
		UserInfoIDs = append(UserInfoIDs, uis[i].ID)
	}
	return UserInfoIDs
}

func CreateUserByPublicKey(publicKeyArmored string, username string, endpoint string) (UserInfo, error) {
	publicKey, err := crypto.NewKeyFromArmored(publicKeyArmored)
	if err != nil {
		log.Println("WARN: Unable to armor public key, returning.")
		return UserInfo{}, errors.New("WARN: Unable to armor public key, returning")
	}
	var ui UserInfo
	DB.Where("fingerprint = ?", publicKey.GetFingerprint()).First(&ui)
	b, err := publicKey.GetArmoredPublicKeyWithCustomHeaders("p3pgo", "")
	if err != nil {
		log.Println("WARN: Unable to publickey.GetPublicKey()")
		return UserInfo{}, errors.New("WARN: Unable to publickey.GetPublicKey()")
	}
	ui.Publickey = b
	ui.Fingerprint = strings.ToLower(publicKey.GetFingerprint())
	ui.KeyID = strings.ToLower(publicKey.GetHexKeyID())
	if username == "" && ui.Username == "" {
		ui.Username = "Unknown User [" + time.Now().String() + "]"
	} else if ui.Username == "" {
		ui.Username = username
	}

	if ui.Endpoint == "" || endpoint != "" {
		ui.Endpoint = Endpoint(endpoint)
	}
	DB.Save(&ui)
	return ui, nil
}
