// +build !go1.6

package graceful

import "crypto/tls"

// see clone16.go
func cloneTLSConfig(cfg *tls.Config) *tls.Config {
	c := *cfg
	return &c
}
