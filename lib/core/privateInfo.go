package core

import (
	"crypto/rand"
	"log"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/ProtonMail/gopenpgp/v2/helper"
	"gorm.io/gorm"
)

type PrivateInfoS struct {
	gorm.Model
	ID          uint
	Username    string
	Bio         string
	PrivateKey  string
	PublicKey   string
	AccountName string
	Passphrase  []byte
	Endpoint    Endpoint
	DB          *gorm.DB `gorm:"-"`
}

func (pi *PrivateInfoS) IsAccountReady() bool {
	pi.Refresh()
	return len(pi.Passphrase) != 0
}
func (pi *PrivateInfoS) Refresh() {
	pi.DB.First(pi)
}
func (pi *PrivateInfoS) Create(username string, email string, bitSize int) {
	pi.DB.FirstOrCreate(pi)
	if len(pi.Passphrase) != 0 {
		log.Fatalln("WARN: Unable to CreatePrivateInfo - because PrivateInfo is not empty.")
		return
	}
	pi.Passphrase = make([]byte, bitSize*16)
	// then we can call rand.Read.
	_, err := rand.Read(pi.Passphrase)
	if err != nil {
		log.Fatalln("Failed to read random data.", err)
	}
	if pi.PrivateKey == "" {
		log.Println("PrivateKey is missing. Generating one.")
		var err error
		pi.PrivateKey, err = helper.GenerateKey(username, email, pi.Passphrase, "rsa", bitSize)
		if err != nil {
			log.Fatalln("Unable to generate privkey:", err)
		}
		pi.DB.Save(&pi)
	}
	privKey, err := crypto.NewKeyFromArmored(pi.PrivateKey)
	if err != nil {
		log.Fatalln("CRIT: Unable to unarmor generated key:", err)
	}

	pubKey, err := privKey.GetArmoredPublicKey()
	if err != nil {
		log.Fatalln("Unable to get armored public key", err)
	}
	pi.PublicKey = pubKey
	pi.Username = username
	pi.DB.Save(&pi)
}
func (pi *PrivateInfoS) Decrypt(armored string) (msg string, keyid string, err error) {
	ciphertext, err := crypto.NewPGPMessageFromArmored(armored)

	if err != nil {
		log.Println("Unable to get signature key id:", err, armored)
		return "", "", err
	}
	privateKeyObj, err := crypto.NewKeyFromArmored(pi.PrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	privateKeyUnlocked, err := privateKeyObj.Unlock(pi.Passphrase)
	if err != nil {
		log.Fatalln(err)
	}
	defer privateKeyUnlocked.ClearPrivateParams()

	privateKeyRing, err := crypto.NewKeyRing(privateKeyUnlocked)
	if err != nil {
		log.Fatalln(err)
	}

	message, err := privateKeyRing.Decrypt(ciphertext, pi.getKeyRing(), 0)
	if err != nil {
		log.Println(err)
		return "", "", nil
	}
	return message.GetString(), pi.findSignFingerprint(ciphertext), nil
}
func (pi *PrivateInfoS) DecryptVerify(armored string, publickey string) (string, error) {
	return helper.DecryptVerifyMessageArmored(publickey, pi.PrivateKey, pi.Passphrase, armored)
}
func (pi *PrivateInfoS) EncryptSign(pubkey string, body string) (string, error) {
	return helper.EncryptSignMessageArmored(pubkey, pi.PrivateKey, pi.Passphrase, body)
}
func (pi *PrivateInfoS) findSignFingerprint(ciphertext *crypto.PGPMessage) string {
	privateKeyObj, err := crypto.NewKeyFromArmored(pi.PrivateKey)
	if err != nil {
		log.Fatalln(err)
	}

	privateKeyUnlocked, err := privateKeyObj.Unlock(pi.Passphrase)
	if err != nil {
		log.Fatalln(err)
	}
	defer privateKeyUnlocked.ClearPrivateParams()

	privateKeyRing, err := crypto.NewKeyRing(privateKeyUnlocked)
	if err != nil {
		log.Fatalln(err)
	}
	var uis []UserInfo
	pi.DB.Find(&uis)
	for i := range uis {
		c, err := crypto.NewKeyRing(nil)
		if err != nil {
			log.Fatalln(err)
		}
		pk, err := crypto.NewKeyFromArmored(uis[i].Publickey)
		if err != nil {
			log.Println(err)
			continue
		}
		err = c.AddKey(pk)
		if err != nil {
			log.Println(err)
			continue
		}
		_, err = privateKeyRing.Decrypt(ciphertext, c, 0)
		if err != nil {
			// log.Println(err)
			continue
		}
		return pk.GetFingerprint()
	}
	return ""
}
func (pi *PrivateInfoS) getKeyRing() *crypto.KeyRing {

	c, err := crypto.NewKeyRing(nil)
	if err != nil {
		log.Fatalln(err)
	}
	var uis []UserInfo
	pi.DB.Find(&uis)
	for i := range uis {
		pk, err := crypto.NewKeyFromArmored(uis[i].Publickey)
		if err != nil {
			log.Println(err)
			continue
		}
		err = c.AddKey(pk)
		if err != nil {
			log.Println(err)
			continue
		}
	}
	return c
}
func (pi *PrivateInfoS) GetDiscoveredUserInfo() DiscoveredUserInfo {
	return DiscoveredUserInfo{
		Name:      pi.Username,
		Bio:       pi.Bio,
		PublicKey: pi.PublicKey,
		Endpoint:  string(pi.Endpoint),
	}
}
