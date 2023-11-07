package core

import (
	"gorm.io/gorm"
)

// We are doing a little rethinking here
// p3p.dart was simply re-encrypting the event,
// this is fine, this isn't time consuming, but only on a
// small scale. Instead we will store it in a simpler way,
// we store the []byte that we need to send, and target
// destination.
// This way we don't rely on anything.
type QueuedEvent struct {
	gorm.Model
	ID       uint
	Body     []byte
	Endpoint Endpoint `json:"endpoints"`
}
