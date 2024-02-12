package core

import (
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"
)

type SharedForBearer struct {
	gorm.Model
	SharedFor string
	Bearer    string
}

type SharedFile struct {
	gorm.Model
	// SharedFor - determines who is allowed to access the file.
	SharedFor     string    `json:"-"`
	Sha512Sum     string    `json:"sha512sum"`
	LastEdit      time.Time `json:"last_edit"`
	FilePath      string    `json:"file_path"`
	SizeBytes     int64     `json:"size_bytes"`
	LocalFilePath string    `json:"-"`
}

func (pi *PrivateInfoS) DeleteSharedFile(sf *SharedFile) {
	err := os.Remove(sf.LocalFilePath)
	if err != nil {
		log.Println(err)
	}
	pi.DB.Delete(sf)
}

func (pi *PrivateInfoS) RemoteFilesAccessBearer(ui *UserInfo) string {
	var sfb SharedForBearer
	pi.DB.First(&sfb, "shared_for = ?", ui.GetKeyID())
	if sfb.Bearer == "" {
		s, err := GenerateRandomStringURLSafe(128)
		if err != nil {
			log.Fatalln("Failed to generate random number", err)
		}
		sfb.Bearer = s
		sfb.SharedFor = ui.GetKeyID()
		pi.DB.Save(&sfb)
	}
	return sfb.Bearer
}

// FileServe - Handle all file requests
// r.Get("/files.http/{sharedFor}/*", FileServe)
func FileServe(w http.ResponseWriter, r *http.Request) {
	sharedFor := chi.URLParam(r, "sharedFor")
	filePath := strings.ReplaceAll(r.RequestURI, fmt.Sprintf("/files.http/%s", sharedFor), "")
	auth := r.Header.Get("Authentication")
	log.Printf("FILE_SERVE(%s): %s: %s [auth: %s]\n", sharedFor, r.RequestURI, filePath, auth)

	pi, err := getPrivateInfoBySharedFor(sharedFor, auth)
	if err != nil {
		w.WriteHeader(403)
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			log.Println(err)
			return
		}
		return
	}

	if filePath == "/.metadata.json" {
		log.Println("\t-> /.metadata.json")
		var lsf []SharedFile
		pi.DB.Find(&lsf, "shared_for = ?", sharedFor)
		m, err := json.MarshalIndent(lsf, "    ", "    ")
		if err != nil {
			log.Fatalln(err)
		}
		_, err = w.Write(m)
		if err != nil {
			log.Println(err)
		}
		return
	}

	var sf SharedFile

	pi.DB.First(&sf, "shared_for = ? AND file_path = ?", sharedFor, filePath)
	if sf.FilePath != filePath || sf.SharedFor != sharedFor {
		w.WriteHeader(404)
		_, err := w.Write([]byte("Unable to find given file"))
		if err != nil {
			log.Println(err)
			return
		}
	}
	http.ServeFile(w, r, sf.LocalFilePath)
}

func getPrivateInfoBySharedFor(sharedFor string, auth string) (*PrivateInfoS, error) {
	if strings.HasPrefix(auth, "Bearer ") {
		auth = auth[len("Bearer "):]
	}
	if sharedFor == "" || auth == "" {
		return nil, errors.New("invalid data provided. No auth or sharedFor")
	}
	for i := range privateInfoMap {
		var sfb SharedForBearer
		privateInfoMap[i].DB.First(&sfb, "shared_for = ? AND bearer = ?", sharedFor, auth)
		if sfb.SharedFor == sharedFor && sfb.Bearer == auth {
			return privateInfoMap[i], nil
		}
	}
	return nil, errors.New("unable to find given pi")
}

func (pi *PrivateInfoS) CreateFile(ui *UserInfo, localFilePath string, remoteFilePath string) error {
	sharedFor := ui.GetKeyID()
	var sf SharedFile
	pi.DB.First(sf, "shared_for = ? AND file_path = ?", sharedFor, remoteFilePath)
	f, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	h := sha512.New()
	_, err = io.Copy(h, f)
	if err != nil {
		return err
	}

	sum := hex.EncodeToString(h.Sum(nil))

	localStorePath := path.Join(pi.StorePath, "files-http", ui.GetKeyID(), sum)

	if fileExists(localStorePath) {
		log.Println("file exists:", localStorePath)
		return errors.New("file with given sha512 checksum already exists")
	}
	//SharedFor     string `json:"-"`
	sf.SharedFor = ui.GetKeyID()
	//Sha512Sum     string `json:"sha512sum"`
	sf.Sha512Sum = sum
	//LastEdit      int    `json:"last_edit"`
	sf.LastEdit = time.Now()
	//FilePath      string `json:"file_path"`
	sf.FilePath = remoteFilePath
	//LocalFilePath string `json:"-"`
	sf.LocalFilePath = localStorePath

	size, err := copyFile(localFilePath, localStorePath)
	if err != nil {
		return err
	}

	sf.SizeBytes = size
	pi.DB.Save(&sf)
	return nil
}

func (pi *PrivateInfoS) GetSharedFiles(ui *UserInfo) (sfs []*SharedFile) {
	sharedFor := ui.GetKeyID()
	pi.DB.Find(sfs, "shared_for = ?", sharedFor)
	return sfs
}

func (pi *PrivateInfoS) GetSharedFilesMetadata(ui *UserInfo) (sfm SharedFilesMetadata) {
	sharedFor := ui.GetKeyID()
	return SharedFilesMetadata{
		FilesEndpoint:  Endpoint(strings.ReplaceAll(fmt.Sprintf("%s/files.http/%s", pi.Endpoint.GetHost(), sharedFor), "//files.http", "/files.http")),
		Authentication: pi.RemoteFilesAccessBearer(ui),
	}
}

func (pi *PrivateInfoS) GetSharedFilesIDs(ui *UserInfo) []uint {
	var sfs []*SharedFile
	sharedFor := ui.GetKeyID()
	pi.DB.Find(&sfs, "shared_for = ?", sharedFor)
	var ints = []uint{}
	for i := range sfs {
		ints = append(ints, sfs[i].ID)
	}
	return ints
}

func (pi *PrivateInfoS) GetSharedFileById(id uint) *SharedFile {
	var sf SharedFile
	pi.DB.First(&sf, "id = ?", id)
	return &sf
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	if err == nil {
		return true
	}
	return false
}

func copyFile(src, dst string) (int64, error) {
	err := os.MkdirAll(dst, 0755)
	if err != nil {
		return 0, err
	}
	err = os.Remove(dst)
	if err != nil {
		return 0, err
	}
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	nBytes, err := io.Copy(destination, source)
	if err != nil {
		return nBytes, err
	}
	err = source.Close()
	if err != nil {
		return nBytes, err
	}
	err = destination.Close()
	if err != nil {
		return nBytes, err
	}
	return nBytes, nil
}
