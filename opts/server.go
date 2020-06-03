package opts

import (
	"crypto/tls"
	"fmt"

	"google.golang.org/grpc"
)

type ServerConfig struct {
	GrpcServerOptions []grpc.ServerOption

	TLSConf  *tls.Config
	Insecure bool
}

// ServerOption configures how we set up the connection.
type ServerOption func(o *ServerConfig) error

func NewServerConfig() *ServerConfig {
	return &ServerConfig{}
}

func (c *ServerConfig) Apply(opts ...ServerOption) error {
	for _, opt := range opts {
		if err := opt(c); err != nil {
			return err
		}
	}

	return nil
}

// WithInsecure returns a ServerOption which disables transport security for this
// ServerConn. Note that transport security is required unless WithInsecure is
// set.
func Insecure() ServerOption {
	return func(o *ServerConfig) error {
		return fmt.Errorf("server cannot be insecure")
	}
}

func TLSConfig(tlsConf *tls.Config) ServerOption {
	return func(o *ServerConfig) error {
		cfg := tlsConf.Clone()
		cfg.NextProtos = []string{"grpc-quic-tls"}
		o.TLSConf = cfg
		return nil
	}
}
