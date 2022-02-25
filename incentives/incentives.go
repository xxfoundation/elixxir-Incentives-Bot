package incentives

import (
	"git.xx.network/elixxir/incentives-bot/storage"
	"gitlab.com/elixxir/client/api"
)

// Impl struct wraps the listener for coupons
type Impl struct {
	*listener
}

// New initializes a listener with passed in storage and client
func New(s *storage.Storage, c *api.Client) *Impl {
	return &Impl{
		&listener{
			s: s,
			c: c,
		},
	}
}
