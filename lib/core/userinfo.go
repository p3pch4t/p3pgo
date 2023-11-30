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
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	Publickey   string   `json:"publickey"`
	Fingerprint string   `json:"-"`
	KeyID       string   `json:"-"`
	Endpoint    Endpoint `json:"endpoint"`
}

func StringToKeyID(str string) string {
	keyid := strings.ToLower(str)
	if len(keyid) > 16 {
		keyid = keyid[len(keyid)-16:]
	}
	return keyid
}

func (ui *UserInfo) GetKeyID() string {
	publicKey, err := crypto.NewKeyFromArmored(ui.Publickey)
	if err != nil {
		log.Panicln(ui.ID, ui.Username, err)
		return ""
	}
	keyid := strings.ToLower(publicKey.GetHexKeyID())
	if len(keyid) > 16 {
		keyid = keyid[len(keyid)-16:]
	}
	return keyid
}

func (ui *UserInfo) SendIntroduceEvent(pi *PrivateInfoS) {
	internalEvent := Event{
		EventType: EventTypeIntroduce,
		Data: EventDataMixed{
			EventDataIntroduce: EventDataIntroduce{
				PublicKey: pi.PublicKey,
				Endpoint:  pi.Endpoint,
				Username:  pi.Username,
			},
		},
	}
	QueueEvent(pi, internalEvent, *ui)
}

func GetUserInfoByID(pi *PrivateInfoS, id uint) (UserInfo, error) {
	var ui UserInfo
	pi.DB.Find(&ui, "id = ?", id)
	if id == 0 || ui.ID != id {
		return UserInfo{ID: id}, errors.New("user with given id couldn't be found")
	}
	return ui, nil
}

func GetUserInfoByKeyID(pi *PrivateInfoS, keyid string) (UserInfo, error) {
	var ui UserInfo
	keyid = StringToKeyID(keyid)
	pi.DB.Find(&ui, "key_id = ?", keyid)
	if keyid == "" || ui.KeyID != keyid {
		return UserInfo{KeyID: keyid}, errors.New("user with given key_id couldn't be found")
	}
	return ui, nil
}

func GetAllUserIDs(pi *PrivateInfoS) (UserInfoIDs []uint) {
	var uis []UserInfo
	pi.DB.Find(&uis)
	for i := range uis {
		if uis[i].Fingerprint == "" && uis[i].Publickey == "" {
			pi.DB.Delete(&uis[i])
			continue
		}
		UserInfoIDs = append(UserInfoIDs, uis[i].ID)
	}
	return UserInfoIDs
}

func CreateUserByPublicKey(pi *PrivateInfoS, publicKeyArmored string, username string, endpoint Endpoint, shouldIntroduce bool) (UserInfo, error) {
	publicKey, err := crypto.NewKeyFromArmored(publicKeyArmored)
	if err != nil {
		log.Println("WARN: Unable to armor public key, returning.")
		return UserInfo{}, errors.New("WARN: Unable to armor public key, returning")
	}
	var ui UserInfo
	pi.DB.Where("fingerprint = ?", publicKey.GetFingerprint()).First(&ui)
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
	pi.DB.Save(&ui)
	if shouldIntroduce {
		ui.SendIntroduceEvent(pi)
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
