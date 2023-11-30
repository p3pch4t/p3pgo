package core

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func InitReachableLocal(pi *PrivateInfoS) {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		b, err := json.Marshal(pi.GetDiscoveredUserInfo())
		if err != nil {
			w.Write([]byte("an error occurred, and response couldn't get generated."))
			return
		}
		w.Write(b)
	})
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		// read body
		b, err := io.ReadAll(r.Body)
		defer r.Body.Close()
		if err != nil && err != io.EOF {
			log.Println("[WARN]: Unable to read:", err)
			w.WriteHeader(500)
			w.Write([]byte("Internal server error"))
			return
		}
		log.Println("processString")
		evts := processString(pi, string(b), "UnKnoWn")
		for i := range evts {
			evts[i].TryProcess(pi)
		}
	})
	go func() {
		err := http.ListenAndServe(":3893", r)
		if err != nil {
			log.Fatalln(err)
		}
	}()
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
