package core

import (
	"log"
	"net/url"
)

type Endpoint string

func (e *Endpoint) GetProtocol() string {
	uri, err := url.Parse(string(*e))
	if err != nil {
		log.Fatalln("Unale to Endpoint.getProtocol:", err)
	}
	return uri.Scheme
}
func (e *Endpoint) GetHost() string {
	uri, err := url.Parse(string(*e))
	if err != nil {
		log.Fatalln("Unale to Endpoint.getHost:", err)
	}
	// ${urip.host}:${urip.port}${urip.path}${urip.query}
	return uri.Host + ":" + uri.Port() + uri.Path + uri.RawQuery
}
func (e *Endpoint) GetExtra() string {
	uri, err := url.Parse(string(*e))
	if err != nil {
		log.Fatalln("Unale to Endpoint.getHost:", err)
	}
	return uri.Fragment
}

/// local or i2p or tor
//String protocol;

/// host - do not think of http header host,
/// let's assume that we are given following profile url
/// local://127.0.0.1:8783/asdfevc?qwer=asd#hashpart
/// ------->127.0.0.1:8783/asdfevc?qwer=asd<--------
/// This would be the host part
//String host;

/// so called "hashpart"
//String extra;
