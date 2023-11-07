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

	pi.PublicKey, err = privKey.GetArmoredPublicKey()
	if err != nil {
		log.Fatalln("Unable to get armored public key", err)
	}
	DB.Save(&pi)
}

func (pi *PrivateInfoS) Decrypt(armored string) (string, error) {
	return helper.DecryptMessageArmored(pi.PrivateKey, pi.Passphrase, armored)
}
func (pi *PrivateInfoS) DecryptVerify(armored string, publickey string) (string, error) {
	return helper.DecryptVerifyMessageArmored(publickey, pi.PrivateKey, pi.Passphrase, armored)
}
func (pi *PrivateInfoS) EncryptSign(pubkey string, body string) (string, error) {
	return helper.EncryptSignMessageArmored(pubkey, pi.PrivateKey, pi.Passphrase, body)
}
