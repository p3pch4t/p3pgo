package core

import (
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"gorm.io/gorm"
)

type UserInfo struct {
	gorm.Model
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	Publickey   string `json:"publickey"`
	Fingerprint string `json:"-"`
	// KeyID       string   `json:"-"`
	Endpoint Endpoint `json:"endpoint"`
}

func (ui *UserInfo) GetKeyID() string {
	publicKey, err := crypto.NewKeyFromArmored(ui.Publickey)
	if err != nil {
		log.Fatalln(ui.ID, ui.Username, err)
		return ""
	}
	keyid := strings.ToLower(publicKey.GetHexKeyID())
	if len(keyid) > 16 {
		keyid = keyid[len(keyid)-16:]
	}
	return keyid
}

func (ui *UserInfo) SendIntroduceEvent() {
	internalEvent := Event{
		EventType: EventTypeIntroduce,
		Data: EventDataMixed{
			EventDataIntroduce: EventDataIntroduce{
				PublicKey: PrivateInfo.PublicKey,
				Endpoint:  PrivateInfo.Endpoint,
				Username:  PrivateInfo.Username,
			},
		},
	}
	QueueEvent(internalEvent,
		*ui)
}

func GetUserInfoByID(id uint) (UserInfo, error) {
	var ui UserInfo
	DB.Find(&ui, "id = ?", id)
	if id == 0 || ui.ID != id {
		return UserInfo{ID: id}, errors.New("user with given id couldn't be found.")
	}
	return ui, nil
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

func CreateUserByPublicKey(publicKeyArmored string, username string, endpoint Endpoint, shouldIntroduce bool) (UserInfo, error) {
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
	if username == "" && ui.Username == "" {
		ui.Username = "Unknown User [" + time.Now().String() + "]"
	} else {
		ui.Username = username
	}

	if ui.Endpoint == "" || endpoint != "" {
		ui.Endpoint = Endpoint(endpoint)
	}
	DB.Save(&ui)
	if shouldIntroduce {
		ui.SendIntroduceEvent()
	}
	return ui, nil
}

type DiscoveredUserInfo struct {
	Name      string `json:"name"`
	Bio       string `json:"bio"`
	PublicKey string `json:"publickey"`
	Endpoint  string `json:"endpoint"`
}

func DiscoverUserByURL(url string) (dui DiscoveredUserInfo, err error) {
	b, err := i2pGet(url)
	if err != nil {
		return DiscoveredUserInfo{}, err
	}
	err = json.Unmarshal(b, &dui)
	if err != nil {
		return dui, err
	}
	return dui, nil
}
