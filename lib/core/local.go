package core

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

var r *chi.Mux

var IsLocalServerRunning = false

var LOCAL_SERVER_PORT = 3893

func StartLocalServer() {
	if IsLocalServerRunning {
		return
	}
	IsLocalServerRunning = true
	r = chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/files.http/{sharedFor}/*", FileServe)
	r.Get("/", getHandleGet())
	r.Post("/", getHandlePost())
	r.Get("/*", getHandleGet())
	r.Post("/*", getHandlePost())
	go func() {
		localServerPort := strconv.Itoa(LOCAL_SERVER_PORT)
		log.Println("starting on :" + localServerPort)
		err := http.ListenAndServe(":"+localServerPort, r)
		if err != nil {
			log.Fatalln(err)
		}
	}()
}

func getPrivateInfoByPath(path string) (pi *PrivateInfoS, pathpart string, err error) {
	if len(path) == 0 {
		path = "/"
	}
	path = path[1:]

	ind := strings.Index(path, "/")
	if ind != -1 {
		path = path[0:ind]
	} else {
		path = path[0:]
	}
	log.Println("path:", path)
	pi, ok := privateInfoMap[path]
	if !ok {
		return &PrivateInfoS{}, path, errors.New("unable to find requested path")
	}

	return pi, path, nil
}

func getHandleGet() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("GET", r.RequestURI)
		pi, pathpart, err := getPrivateInfoByPath(r.RequestURI)
		if err != nil {
			log.Println(err)
			w.WriteHeader(500)
			_, err := w.Write([]byte(err.Error()))
			if err != nil {
				log.Println(err)
			}
			return
		}
		log.Println(pathpart, r.RequestURI)
		// Are we looking for a file?
		// we are not looking for a file

		processDiscovery(pi, w, r)

	}
}

func processDiscovery(pi *PrivateInfoS, w http.ResponseWriter, r *http.Request) {
	b, err := json.Marshal(pi.GetDiscoveredUserInfo())
	if err != nil {
		w.WriteHeader(500)
		_, err := w.Write([]byte("an error occurred, and response couldn't get generated."))
		if err != nil {
			log.Println(err)
		}
		return
	}
	_, err = w.Write(b)
	if err != nil {
		log.Println(err)
	}
}

func getHandlePost() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		pi, _, err := getPrivateInfoByPath(r.RequestURI)
		if err != nil {
			_, err := w.Write([]byte(err.Error()))
			if err != nil {
				log.Println(err)
			}
			return
		}
		b, err := io.ReadAll(r.Body)
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Println(err)
			}
		}(r.Body)
		if err != nil && err != io.EOF {
			log.Println("[WARN]: Unable to read:", err)
			w.WriteHeader(500)
			_, err := w.Write([]byte("Internal server error"))
			if err != nil {
				log.Println(err)
			}
			return
		}
		log.Println("processString")
		evts := processString(pi, string(b), "UnKnoWn")
		for i := range evts {
			evts[i].TryProcess(pi)
		}
	}
}

var privateInfoMap = make(map[string]*PrivateInfoS)

func (pi *PrivateInfoS) InitReachableLocal(path string) {
	privateInfoMap[path] = pi
}

func processString(pi *PrivateInfoS, evt string, keyid string) (evts []Event) {
	//log.Println("str:", evt)
	// json decode
	var tmpDecode Event
	err0 := json.Unmarshal([]byte(evt), &evts)
	err1 := json.Unmarshal([]byte(evt), &tmpDecode)
	if err1 == nil && tmpDecode.Uuid != "" {
		evts = append(evts, tmpDecode)
	}

	// assume plaintext

	if err0 != nil && err1 != nil {
		// We have failed to unmarshal them, let's decrypt them
		str, _keyid, err := pi.Decrypt(evt)
		keyid = _keyid
		log.Println("keyid:", keyid)
		for i := range evts {
			evts[i].InternalKeyID = keyid
		}
		if err != nil {
			log.Println(err)
			// malformed or encrypted with different publickey.
			return evts
		}

		return append(evts, processString(pi, str, keyid)...)
	}
	for i := range evts {
		evts[i].InternalKeyID = keyid
	}
	// log.Println("processString: evts:", evts)
	return evts
}
