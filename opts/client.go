package opts

import (
	"crypto/tls"

	"google.golang.org/grpc"
)

type ClientConfig struct {
	GrpcDialOptions []grpc.DialOption

	TLSConf  *tls.Config
	Insecure bool
}

// DialOption configures how we set up the connection.
type DialOption func(o *ClientConfig) error

func NewClientConfig() *ClientConfig {
	return &ClientConfig{}
}

func (c *ClientConfig) Apply(opts ...DialOption) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return err
		}
	}

	return nil
}

// WithInsecure returns a DialOption which disables transport security for this
// ClientConn. Note that transport security is required unless WithInsecure is
// set.
func WithInsecure() DialOption {
	return func(o *ClientConfig) error {
		o.Insecure = true
		o.TLSConf = &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"grpc-quic-tls"}}
		return nil
	}
}

func WithTLSConfig(tlsConf *tls.Config) DialOption {
	return func(o *ClientConfig) error {
		cfg := tlsConf.Clone()
		cfg.NextProtos = []string{"grpc-quic-tls"}
		o.TLSConf = cfg
		return nil
	}
}
