package core

import (
	"gorm.io/gorm"
)

type UserInfo struct {
	gorm.Model
	ID          uint     `json:"id"`
	Username    string   `json:"username"`
	Publickey   string   `json:"publickey"`
	Fingerprint string   `json:"_GO_fingerprint"`
	Endpoint    Endpoint `json:"endpoint"`
}
