package core

import (
	"crypto/sha512"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"time"

	UUID "github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateFileStoreElement Creates or Updates FileStoreElement
func CreateFileStoreElement(uiKeyId string, uuid string, path string, localFilePath string, modifyTime int64) FileStoreElement {
	log.Println(
		"CreateFileStoreElement(", "\n",
		"\tuiKeyId:", uiKeyId, "\n",
		"\tuuid:", uuid, "\n",
		"\tpath:", path, "\n",
		"\tlocalFilePath:", localFilePath, "\n",
		"\tmodifyTime:", modifyTime, "\n",
		")",
	)
	var fi = FileStoreElement{}
	var sizeBytes int64 = 0
	// Since we are creating a new file it's sha512sum is equal to sha512sum of /dev/null
	// To avoid calculating the file sha512sum and to actually allow new files to be created - we need something like
	// this
	sha512sum := "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
	if uuid == "" {
		uuid = UUID.New().String()
	} else {
		DB.Find(&fi, "uuid = ?", uuid)
	}

	if path == "" {
		path = "/Unsort/" + time.Now().String()
	}

	fi.InternalKeyID = uiKeyId
	fi.Uuid = uuid
	fi.Path = path
	fi.Sha512sum = sha512sum
	fi.SizeBytes = sizeBytes
	fi.ModifyTime = modifyTime

	if localFilePath != fi.LocalPath() && localFilePath != "" {
		storeFile := fi.GetFile()
		f, err := os.Open(localFilePath)
		if err != nil {
			log.Println(err)
		} else {
			_, err = io.Copy(storeFile, f)
			if err != nil {
				log.Fatalln(err)
			}
		}
	}

	f := fi.GetFile()
	sha_512 := sha512.New()
	var err error
	fi.SizeBytes, err = io.Copy(sha_512, f)
	if err != nil {
		log.Fatalln(err)
	}
	fi.Sha512sum = fmt.Sprintf("%x", sha_512.Sum(nil))

	DB.Save(&fi)
	return fi
}

func GetFileStoreById(id uint) (FileStoreElement, error) {
	var fse FileStoreElement
	DB.First(&fse, "ID = ?", id)
	if fse.ID != id {
		return fse, errors.New("unable to find given FileStoreElement")
	}
	return fse, nil
}

type FileStoreElement struct {
	gorm.Model
	InternalKeyID string `json:"-"`
	Uuid          string `json:"uuid,omitempty"`
	//Path - is the in chat path, eg /Apps/Calendar.xdc
	Path string `json:"path,omitempty"`
	//LocalPath - is the filesystem path
	Sha512sum     string `json:"sha512sum,omitempty"`
	SizeBytes     int64  `json:"sizeBytes,omitempty"`
	IsDeleted     bool   `json:"isDeleted,omitempty"`
	IsDownloading bool   `json:"-"`
	ModifyTime    int64  `json:"modifyTime,omitempty"`
}

func (fse *FileStoreElement) IsDownloaded() bool {
	f := fse.GetFile()
	fi, err := f.Stat()
	if err != nil {
		log.Fatalln(err)
	}
	if fse.SizeBytes == fi.Size() {
		return true
	}
	return false
}

func (fse *FileStoreElement) Refresh(ui UserInfo) {
	var fseNew FileStoreElement
	DB.Find(&fseNew, "uuid = ? AND internal_key_id = ?", fse.Uuid, ui.GetKeyID())
	fse.InternalKeyID = fseNew.InternalKeyID
	if fse.LocalPath() == "" {
		f := fse.GetFile()
		err := f.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func (fse *FileStoreElement) UpdateContent(announce bool) {
	if fse.IsDownloading {
		log.Println("fse.UpdateContent() called when .IsDownloading == true. Don't do that.")
		return
	}
	CreateFileStoreElement(fse.InternalKeyID, fse.Uuid, fse.Path, fse.LocalPath(), time.Now().UnixMicro())
	if !announce {
		return
	}
	fse.Announce()
}

func (fse *FileStoreElement) Announce() {
	ui, err := GetUserInfoByKeyID(fse.InternalKeyID)
	if err != nil {
		log.Fatalln(err)
	}

	b, err := io.ReadAll(fse.GetFile())
	if err != nil {
		log.Fatalln(err)
	}
	QueueEvent(Event{
		InternalKeyID: ui.GetKeyID(),
		EventType:     EventTypeFile,
		Data: EventDataMixed{
			EventDataFile: EventDataFile{
				Uuid:       fse.Uuid,
				Bytes:      b,
				Path:       fse.Path,
				Sha512sum:  fse.Sha512sum,
				SizeBytes:  fse.SizeBytes,
				IsDeleted:  false,
				ModifyTime: fse.ModifyTime,
			},
		},
		Uuid: "",
	}, ui)
}

func (fse *FileStoreElement) LocalPathDir() string {
	fpdir := path.Join(storePath, "filstore", fse.InternalKeyID)
	_, err := os.Stat(fpdir)
	if err != nil {
		err := os.MkdirAll(fpdir, 0750)
		if err != nil {
			log.Fatalln(err)
		}
	}
	return fpdir
}
func (fse *FileStoreElement) LocalPath() string {
	fpath := path.Join(fse.LocalPathDir(), fse.Uuid)

	_, err := os.Stat(fpath)
	if err != nil {
		f, err := os.Create(fpath)
		if err != nil {
			log.Fatalln(err)
		}
		f.Sync()
		err = f.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
	return fpath
}

func (fse *FileStoreElement) GetFile() *os.File {
	if fse.InternalKeyID == "" {
		log.Fatalln("fse.InternalKeyID is empty. Did you forget to fse.Refresh(ui)?")
	}
	fpfile := fse.LocalPath()
	f, err := os.OpenFile(fpfile, os.O_RDWR, 0750)
	if err != nil {
		log.Fatalln(err)
	}
	return f
}

func fileStoreElementQueueRunner() {
	for {
		var felms []FileStoreElement
		DB.Find(&felms)
		for i := range felms {
			if felms[i].IsDeleted {
				// We don't care to update deleted files.
				continue
			}
			fi, err := felms[i].GetFile().Stat()
			if err != nil {
				log.Fatalln(err)
			}
			if fi.Size() == felms[i].SizeBytes {
				// All is fine. The file is exactly the same size on file system and
				// in database
				continue
			}
			// In this case, we are supposed to push an update to the UserInfo
			felms[i].UpdateContent(true)
		}
		time.Sleep(time.Second * 5)
	}
}
