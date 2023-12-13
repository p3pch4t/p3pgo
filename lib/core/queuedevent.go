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
	RelayTries  int
	LastRelayed time.Time
	Body        []byte
	Endpoint    Endpoint
}

func (evt *QueuedEvent) Relay(pi *PrivateInfoS) error {
	evt.LastRelayed = time.Now()
	evt.RelayTries++
	pi.DB.Save(evt)
	host := evt.Endpoint.GetHost()
	if host == "" || host == "http://:" {
		log.Println("Removed event from queue:", evt.ID, "reason: host is not found")
		pi.DB.Delete(evt)
		return errors.New("host is empty - removed queued event")
	}
	_, err := i2pPost(host, evt.Body)
	if err != nil {
		log.Println(err)
		// DB.Delete(evt)
		return err
	}
	pi.DB.Delete(evt)
	return nil
}

func GetQueuedEvents(pi *PrivateInfoS) (evts []*QueuedEvent) {
	pi.DB.Order("RANDOM()").Limit(50).Find(&evts)
	return evts
}

var I2P_HTTP_PROXY = "http://127.0.0.1:4444"

func i2pHttpTransport() *http.Transport {
	proxyUrl, err := url.Parse(I2P_HTTP_PROXY)
	if err != nil {
		log.Fatalln(err)
	}
	return &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
}

func i2pPost(uri string, body []byte) ([]byte, error) {
	httpClient := &http.Client{Transport: i2pHttpTransport(), Timeout: time.Second * 60}
	// log.Println("Body:" + string(body))
	req, err := http.NewRequest("POST", uri, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")
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
		return []byte{}, errors.New("unknown server response")
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Println("Failed to .Close()", err)
		}
	}(respbody.Body)
	b, err := io.ReadAll(respbody.Body)
	if err != nil {
		log.Println(err)
		return b, err
	}
	log.Println("OK:", string(b))
	return b, nil
}

func i2pGet(uri string) ([]byte, error) {
	httpClient := &http.Client{Transport: i2pHttpTransport(), Timeout: time.Second * 14}
	// log.Println("Body:" + string(body))
	req, err := http.NewRequest("GET", uri, nil)
	req.Header.Set("Content-Type", "application/octet-stream")
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
		return []byte{}, errors.New("unknown server response")
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

var queueTimeout = make(map[string]int)

func (pi *PrivateInfoS) EventQueueRunner() {
	for {
		var emptyList []Endpoint
	OuterLoop:
		for _, evt := range GetQueuedEvents(pi) {
			for i := range emptyList {
				if evt.Endpoint == emptyList[i] {
					break OuterLoop
				}
			}
			_, ok := queueTimeout[string(evt.Endpoint)]
			if !ok {
				queueTimeout[string(evt.Endpoint)] = 0
			}
			emptyList = append(emptyList, evt.Endpoint)
			if queueTimeout[string(evt.Endpoint)] > 0 {
				queueTimeout[string(evt.Endpoint)]--
				continue
			}

			queueTimeout[string(evt.Endpoint)] = 5
			log.Println("processing event:", evt.ID)
			go evt.Relay(pi)
		}
		time.Sleep(time.Second * 1)
	}
}
