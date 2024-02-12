package core

import (
	"bytes"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync"
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

func (evt *QueuedEvent) GetEndpointStats(pi *PrivateInfoS) *EndpointStats {
	return pi.getEndpointStats(evt.Endpoint)
}

func (evt *QueuedEvent) Relay(pi *PrivateInfoS) error {
	evt.LastRelayed = time.Now()
	evt.RelayTries++
	pi.DB.Save(evt)
	es := evt.GetEndpointStats(pi)
	if !es.ShouldRelayNow(pi) {
		return errors.New("es.ShouldRelayNow says we shouldn't relay it")
	}
	host := evt.Endpoint.GetHost()
	if host == "" || host == "http://:" || host == "http://" {
		log.Println("Removed event from queue:", evt.ID, "reason: host is not found")
		pi.DB.Delete(evt)
		return errors.New("host is empty - removed queued event")
	}
	_, err := i2pPost(host, evt.Body)
	if err != nil {
		// DB.Delete(evt)
		es.Fail(pi)
		return err
	}
	es.SuccessOut(pi)
	pi.DB.Delete(evt)
	return nil
}

func (pi *PrivateInfoS) getEndpointStats(endpoint Endpoint) *EndpointStats {
	var endpointStats EndpointStats
	pi.DB.First(&endpointStats, "endpoint = ?", string(endpoint))
	endpointStats.Endpoint = string(endpoint)
	return &endpointStats
}

func (pi *PrivateInfoS) GetEndpointStatsByID(id int) *EndpointStats {
	var endpointStats EndpointStats
	pi.DB.First(&endpointStats, "id = ?", id)
	return &endpointStats
}

type EndpointStats struct {
	gorm.Model
	Endpoint       string
	LastContactOut time.Time
	// LastContactIn  time.Time
	// We need some specific way of defining when to contact a user.
	// I assume that we want to try frequently for first two days,
	// and if after 48 hours we hear no reply skip contacting that user.
	// Also by 48 hours I mean 48 hours of trying. If we suddenly go
	// offline we should resume our timer right where it was.
	// Retries should be spread across 48 hours differently:
	// so to come up with something simple:
	// | Count | Delay | Time | Total Time | Total Reqs |
	// | ----- | ----- | ---- | ---------- | ---------- |
	// | 60    | 15s   | 15m  | 15m        | 60         |
	// | 90    | 30s   | 45m  | 1h         | 150        |
	// | 120   | 1m    | 2h   | 3h         | 270        |
	// | 90    | 2m    | 3h   | 6h         | 360        |
	// | 216   | 5m    | 18h  | 1d         | 576        |
	// | 144   | 10m   | 1d   | 2d         | 720        |
	// With table like this we should spend 48hours on trying to reach
	// given user with at most 720 request in total.

	// FailInRow should be reset to 0 when we manage to contact given
	// endpoint or in case when we will be contacted by given endpoint,
	// we should reset it back to 60
	FailInRow int

	// CurrentDelay stores information about how much time should we spend
	// before trying to reach this endpoint again.
	CurrentDelay int
}

func (es *EndpointStats) Fail(pi *PrivateInfoS) {
	if es.FailInRow < 0 {
		es.FailInRow = 0
	} else {
		es.FailInRow++
	}
	pi.DB.Save(es)
}

func (es *EndpointStats) SuccessOut(pi *PrivateInfoS) {
	es.LastContactOut = time.Now()
	if es.FailInRow > 0 {
		es.FailInRow = 0
	} else {
		es.FailInRow--
	}
	pi.DB.Save(es)
}

//func (es *EndpointStats) SuccessIn(pi *PrivateInfoS) {
//	es.LastContactIn = time.Now()
//	es.FailInRow = 60
//	pi.DB.Save(es)
//}

func (es *EndpointStats) ShouldRelayNow(pi *PrivateInfoS) bool {
	if es.FailInRow <= 0 {
		es.CurrentDelay = 0
		pi.DB.Save(es)
		return true
	}
	if es.CurrentDelay >= 0 {
		log.Println("currentDelay: ", es.CurrentDelay)
		es.CurrentDelay--
		pi.DB.Save(es)
		return false
	}

	rules := []struct {
		Count int
		Delay int
	}{
		{60, 15},
		{90, 30},
		{120, 1 * 60},
		{90, 2 * 60},
		{216, 5 * 60},
		{144, 10 * 60},
	}

	curc := es.FailInRow

	delay := 0

	for _, rule := range rules {
		if curc < rule.Count {
			delay = rule.Delay
		}
		curc -= rule.Count
	}

	es.CurrentDelay = delay
	pi.DB.Save(es)
	log.Println("es.FailInRow:", es.FailInRow, "curc", curc, "delay", delay)

	return false
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
var queueLock = make(map[string]*sync.Mutex)

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
			_, ok := queueLock[string(evt.Endpoint)]
			if !ok {
				queueLock[string(evt.Endpoint)] = &sync.Mutex{}
			}
			_, ok = queueTimeout[string(evt.Endpoint)]
			if !ok {
				queueTimeout[string(evt.Endpoint)] = 0
			}

			emptyList = append(emptyList, evt.Endpoint)
			if queueTimeout[string(evt.Endpoint)] > 0 {
				queueTimeout[string(evt.Endpoint)]--
				continue
			}
			go func(evt *QueuedEvent) {
				if !queueLock[string(evt.Endpoint)].TryLock() {
					return
				}
				log.Println("processing event:", evt.ID)
				err := evt.Relay(pi)
				if err != nil {
					log.Println("Failed to relay event:", err)
					queueTimeout[string(evt.Endpoint)] = 60
				}
				queueLock[string(evt.Endpoint)].Unlock()
			}(evt)
		}
		time.Sleep(time.Second * 1)
	}
}
