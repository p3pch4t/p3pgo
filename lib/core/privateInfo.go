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
	ID         uint
	Username   string
	PrivateKey string
	PublicKey  string
	Passphrase []byte
}

var PrivateInfo = PrivateInfoS{ID: 0}

func (pi *PrivateInfoS) Refresh() {
	DB.FirstOrCreate(pi)
	if len(pi.Passphrase) == 0 {
		pi.Passphrase = make([]byte, 4096)
		// then we can call rand.Read.
		_, err := rand.Read(pi.Passphrase)
		if err != nil {
			log.Fatalln("Failed to read random data.", err)
		}
	}
	if pi.PrivateKey == "" {
		log.Println("PrivateKey is missing. Generating one.")
		var err error
		pi.PrivateKey, err = helper.GenerateKey("placeholder name", "user@example.com", pi.Passphrase, "rsa", 4096)
		if err != nil {
			log.Fatalln("Unable to generate privkey:", err)
		}
		DB.Save(&pi)
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
	pi.Username = privKey.GetFingerprint()
	DB.Save(&pi)
}

func (pi *PrivateInfoS) Decrypt(armored string) (msg string, keyid string, err error) {
	ciphertext, err := crypto.NewPGPMessageFromArmored(armored)

	if err != nil {
		log.Fatalln("Unable to get signature key id:", err)
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
		log.Fatalln(err)
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
	DB.Find(&uis)
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
			log.Println(err)
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
	DB.Find(&uis)
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
