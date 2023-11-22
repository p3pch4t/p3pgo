package core

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"gorm.io/gorm"

	"moul.io/http2curl"
)

// QueuedEvent - We are doing a little rethinking here
// p3p.dart was simply re-encrypting the event,
// this is fine, this isn't time-consuming, but only on a
// small scale. Instead, we will store it in a simpler way,
// we store the []byte that we need to send, and target
// destination.
// This way we don't rely on anything.
type QueuedEvent struct {
	gorm.Model
	ID       uint
	Body     []byte
	Endpoint Endpoint `json:"endpoints"`
}

func (evt *QueuedEvent) Relay() {
	host := evt.Endpoint.GetHost()
	if host == "" || host == "http://:" {
		log.Println("Removed event from queue:", evt.ID, "reason: host is not found")
		DB.Delete(evt)
		return
	}
	_, err := i2pPost(host, evt.Body)
	if err != nil {
		log.Println(err)
		return
	}
	DB.Delete(evt)

}

func GetQueuedEvents() (evts []QueuedEvent) {
	DB.Order("RANDOM()").Limit(50).Find(&evts)
	return evts
}

func i2pPost(uri string, body []byte) ([]byte, error) {
	proxyUrl, err := url.Parse("http://127.0.0.1:4444")
	httpClient := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyUrl)}, Timeout: time.Second * 30}
	// log.Println("Body:" + string(body))
	req, err := http.NewRequest("POST", uri, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octent-stream")
	if err != nil {
		return []byte{}, err
	}
	_, err = http2curl.GetCurlCommand(req)
	if err != nil {
		log.Fatalln(err)
	}
	respbody, err := httpClient.Do(req)
	if err != nil {
		return []byte{}, err
	}
	if respbody.StatusCode != 200 {
		return []byte{}, errors.New("unknown server response.")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Failed to .Close()", err)
		}
	}(respbody.Body)
	b, err := io.ReadAll(respbody.Body)
	log.Println("OK:", string(b))
	return b, nil
}
func queueRunner() {
	for {
		log.Println("queueRunner:")
		for _, evt := range GetQueuedEvents() {
			evt.Relay()
		}
		time.Sleep(time.Second * 5)
	}
}
