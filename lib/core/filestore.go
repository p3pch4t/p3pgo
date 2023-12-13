package core

import (
	"crypto/sha512"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/cavaliergopher/grab/v3"
	UUID "github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateFileStoreElement Creates or Updates FileStoreElement (if uikeyid/uuid combo exists)
func (pi *PrivateInfoS) CreateFileStoreElement(uiKeyId string, uuid string, path string, localFilePath string, modifyTime int64, httpPath string) FileStoreElement {
	log.Println(
		"CreateFileStoreElement(", "\n",
		"\tuiKeyId:", uiKeyId, "\n",
		"\tuuid:", uuid, "\n",
		"\tpath:", path, "\n",
		"\tlocalFilePath:", localFilePath, "\n",
		"\tmodifyTime:", modifyTime, "\n",
		"\thttpPath:", httpPath, "\n",
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
		pi.DB.Find(&fi, "uuid = ?", uuid)
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
	ui, err := pi.GetUserInfoByKeyID(uiKeyId)
	if err != nil {
		log.Fatalln(err)
	}

	// The file is provided, so we copy it from the localFilePath to
	// local app database.
	if localFilePath != fi.LocalPath() && localFilePath != "" {
		fi.ExternalHttpPath = ""
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

	if httpPath != "" {
		if strings.HasPrefix(httpPath, "http://") || strings.HasPrefix(httpPath, "https://") {
			fi.ExternalHttpPath = httpPath
		} else {
			fi.ExternalHttpPath = ui.Endpoint.GetHost() + "/" + httpPath
		}
		if !pi.IsMini {
			_ = fi.GetFile().Truncate(0) // We truncate it just in case
		}
	}

	if localFilePath != "" {
		f := fi.GetFile()
		sha512sum := sha512.New()
		fi.SizeBytes, err = io.Copy(sha512sum, f)
		if err != nil {
			log.Fatalln(err)
		}
		fi.Sha512sum = fmt.Sprintf("%x", sha512sum.Sum(nil))
	}
	b, _ := json.MarshalIndent(fi, "", "    ")
	log.Println(string(b))
	pi.DB.Save(&fi)
	return fi
}

func (pi *PrivateInfoS) GetFileStoreById(id uint) (FileStoreElement, error) {
	var fse FileStoreElement
	pi.DB.First(&fse, "ID = ?", id)
	if fse.ID != id {
		return fse, errors.New("unable to find given FileStoreElement")
	}
	return fse, nil
}

type FileStoreElement struct {
	gorm.Model
	InternalKeyID    string `json:"-"`
	Uuid             string `json:"uuid,omitempty"`
	ExternalHttpPath string `json:"externalHttpPath,omitempty"`
	//Path - is the in chat path, eg /Apps/Calendar.xdc
	Path string `json:"path,omitempty"`
	//LocalPath - is the filesystem path
	Sha512sum     string `json:"sha512sum,omitempty"`
	SizeBytes     int64  `json:"sizeBytes,omitempty"`
	IsDeleted     bool   `json:"isDeleted,omitempty"`
	IsDownloading bool   `json:"-"`
	ModifyTime    int64  `json:"modifyTime,omitempty"`
}

func (fse *FileStoreElement) fsSha512() string {
	f := fse.GetFile()

	sha_512 := sha512.New()
	var err error
	_, err = io.Copy(sha_512, f)
	if err != nil {
		log.Fatalln(err)
	}
	return fmt.Sprintf("%x", sha_512.Sum(nil))
}

func (fse *FileStoreElement) IsDownloaded() bool {
	f := fse.GetFile()
	fi, err := f.Stat()
	if err != nil {
		log.Fatalln(err)
	}
	// Don't calculate checksum if we are obviously different
	if fse.SizeBytes == fi.Size() {
		return fse.Sha512sum == fse.fsSha512()
	}
	return false
}

func (fse *FileStoreElement) Refresh(pi *PrivateInfoS, ui UserInfo) {
	var fseNew FileStoreElement
	pi.DB.Find(&fseNew, "uuid = ? AND internal_key_id = ?", fse.Uuid, ui.GetKeyID())
	fse.InternalKeyID = fseNew.InternalKeyID
	if fse.LocalPath() == "" {
		f := fse.GetFile()
		err := f.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func (fse *FileStoreElement) UpdateContent(pi *PrivateInfoS, announce bool) {
	log.Println("UpdateContent")
	if fse.IsDownloading {
		log.Println("fse.UpdateContent() called when .IsDownloading == true. Don't do that.")
		return
	}
	pi.CreateFileStoreElement(fse.InternalKeyID, fse.Uuid, fse.Path, fse.LocalPath(), time.Now().UnixMicro(), "")
	if !announce {
		return
	}
	fse.Announce(pi)
}

func (fse *FileStoreElement) Announce(pi *PrivateInfoS) {
	ui, err := pi.GetUserInfoByKeyID(fse.InternalKeyID)
	if err != nil {
		log.Println(fse.InternalKeyID)
		log.Fatalln(err)
	}

	// b, err := io.ReadAll(fse.GetFile())
	// if err != nil {
	// 	log.Fatalln(err)
	// }
	QueueEvent(pi, Event{
		InternalKeyID: ui.GetKeyID(),
		EventType:     EventTypeFile,
		Data: EventDataMixed{
			EventDataFile: EventDataFile{
				Uuid:       fse.Uuid,
				HttpPath:   fse.HttpRequestPart(),
				Path:       fse.Path,
				Sha512sum:  fse.fsSha512(),
				SizeBytes:  fse.SizeBytes,
				IsDeleted:  fse.IsDeleted,
				ModifyTime: fse.ModifyTime,
			},
		},
		Uuid: "",
	}, ui)
}

func (fse *FileStoreElement) HttpRequestPart() string {
	// Instead of relying on Uuid Sha512sum may be better,
	// in future I may want to implement something like keybase's
	// virtual directories when you can `cd` into the past, to
	// see what files were shared there. In groups this could
	// be a nice feature.
	return fse.InternalKeyID + "/" + fse.fsSha512()
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
	fsefile := fse.LocalPath()
	f, err := os.OpenFile(fsefile, os.O_RDWR, 0750)
	if err != nil {
		log.Fatalln(err)
	}
	return f
}

func (pi *PrivateInfoS) FileStoreElementQueueRunner() {
	for {
		var felms []FileStoreElement
		pi.DB.Find(&felms)
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
			if felms[i].IsDownloading {
				continue
			}
			// In this case, we are supposed to push an update to the UserInfo
			felms[i].UpdateContent(pi, true)
		}
		time.Sleep(time.Second * 5)
	}
}

var fsedlMutex = make(map[string]*sync.Mutex)

func (pi *PrivateInfoS) FileStoreElementDownloadLoop() {
	for {
		var fselist []FileStoreElement
		pi.DB.Not("external_http_path = \"\" AND is_deleted = true").Find(&fselist)
		for i := range fselist {
			// log.Println("fseDlLoop", fselist[i].ID, fselist[i].Path, fselist[i].ExternalHttpPath)
			go fselist[i].downloadSafe(pi)
		}
		time.Sleep(time.Second * 5)
	}
}

func (fse *FileStoreElement) getShortKey() string {
	key := fse.Sha512sum + "_" + fse.Uuid + "_" + fse.InternalKeyID
	return GetMD5Hash(key)
}

func (fse *FileStoreElement) getMutex() *sync.Mutex {
	key := fse.getShortKey()
	_, ok := fsedlMutex[key]
	if !ok {
		fsedlMutex[key] = &sync.Mutex{}
	}
	return fsedlMutex[key]
}

func (fse *FileStoreElement) downloadSafe(pi *PrivateInfoS) {
	if pi.IsMini {
		log.Println("WARN: downloadSafe() called while isMini == true")
		return
	}
	if fse.ExternalHttpPath == "" {
		return
	}
	mut := fse.getMutex()
	// I do believe that it is the correct use of .TryLock,
	// despite the fact that comment on this function:
	// > Note that while correct uses of TryLock do exist,
	// > they are rare, and use of TryLock is often a sign
	// > of a deeper problem in a particular use of mutexes.
	// got me thinking here for a pretty long time, if someone
	// is aware of a better logic here, please let me know
	if !mut.TryLock() {
		return
	}
	fse.IsDownloading = true
	pi.DB.Save(fse)

	client := grab.NewClient()
	client.HTTPClient = &http.Client{Transport: i2pHttpTransport()}
	// client.HTTPClient = i2pHttpTransport()
	_ = os.Remove(fse.LocalPath())

	tries := 0
OuterLoop:
	for {
		tries++
		if tries > 15 {
			pi.DB.Delete(&fse)
			mut.Unlock()
			return
		}
		req, _ := grab.NewRequest(fse.LocalPath(), fse.ExternalHttpPath)
		log.Printf("Downloading %d, %v...\n", tries, req.URL())
		resp := client.Do(req)
		t := time.NewTicker(time.Second)
		defer t.Stop()

		for {
			select {
			case <-t.C:
				fmt.Printf("%.02f%% complete\n", resp.Progress())
				continue OuterLoop

			case <-resp.Done:
				if err := resp.Err(); err != nil {
					log.Println(err)
				}
				// file downloaded
				//TODO: merge 2 .xdc.update.jsonp files.
				log.Println("File downloaded!")
				fse.IsDownloading = false
				fse.ExternalHttpPath = ""
				pi.DB.Save(fse)
				ui, err := pi.GetUserInfoByKeyID(fse.InternalKeyID)
				if err != nil {
					log.Println(err)
					break OuterLoop
				}
				fse.UpdateContent(pi, false)
				for i := range pi.FileStoreElementCallback {
					pi.FileStoreElementCallback[i](pi, ui, fse, true)
				}
				break OuterLoop
			}
		}
	}
	mut.Unlock()
}
