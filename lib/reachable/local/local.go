package reachable_local

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/core"
	"git.mrcyjanek.net/p3pch4t/p3pgo/lib/events"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
func InitReachableLocal() {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("p3p.go"))
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

		evts := processString(string(b), "")
		for i := range evts {
			evts[i].TryProcess()
		}
	})
	go http.ListenAndServe(":3000", r)
}

func processString(evt string, keyid string) (evts []events.Event) {
	// json decode
	var tmpDecode events.Event
	err0 := json.Unmarshal([]byte(evt), &evts)
	err1 := json.Unmarshal([]byte(evt), &tmpDecode)
	if err1 == nil && tmpDecode.Uuid != "" {
		evts = append(evts, tmpDecode)
	}
	// assume plaintext

	if err0 != nil && err1 != nil {
		// We have failed to decode them, let's decrypt them
		str, keyid, err := core.PrivateInfo.Decrypt(evt)
		log.Println("keyid:", keyid)
		if err != nil {
			// malformed or encrypted with different publickey.
			return evts
		}
		return append(evts, processString(str, keyid)...)
	}

	return
}
