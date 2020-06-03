package grpcquic

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	qnet "github.com/bodaay/grpc-quic/net"
	options "github.com/bodaay/grpc-quic/opts"
	"github.com/bodaay/grpc-quic/transports"
	quic "github.com/lucas-clemente/quic-go"
	"google.golang.org/grpc"
)

var quicConfig = &quic.Config{
	// MaxReceiveStreamFlowControlWindow:     3 * (1 << 20),   // 3 MB
	// MaxReceiveConnectionFlowControlWindow: 4.5 * (1 << 20), // 4.5 MB
	// Versions: []quic.VersionNumber{101},
	// AcceptCookie: func(clientAddr net.Addr, cookie *quic.Cookie) bool {
	// 	// TODO(#6): require source address validation when under load
	// 	return true
	// },
	KeepAlive: true,
}

func newPacketConn(addr string) (net.PacketConn, error) {
	// create a packet conn for outgoing connections
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	return net.ListenUDP("udp", udpAddr)
}

func newQuicDialer(tlsConf *tls.Config) func(string, time.Duration) (net.Conn, error) {
	return func(target string, timeout time.Duration) (net.Conn, error) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		sess, err := quic.DialAddrContext(ctx, target, tlsConf, quicConfig)
		if err != nil {
			return nil, err
		}

		return qnet.NewConn(sess)
	}
}

func Dial(addr string, opts ...options.DialOption) (*grpc.ClientConn, error) {
	cfg := options.NewClientConfig()
	if err := cfg.Apply(opts...); err != nil {
		return nil, err
	}
	creds := transports.NewCredentials(cfg.TLSConf)
	dialer := newQuicDialer(cfg.TLSConf)
	grpcOpts := []grpc.DialOption{
		grpc.WithDialer(dialer),
		grpc.WithTransportCredentials(creds),
	}
	grpcOpts = append(grpcOpts, cfg.GrpcDialOptions...)
	return grpc.Dial(addr, grpcOpts...)
}

func newListener(addr string, tlsConf *tls.Config) (net.Listener, error) {
	pconn, err := newPacketConn(addr)
	if err != nil {
		return nil, err
	}
	ql, err := quic.Listen(pconn, tlsConf, quicConfig)
	if err != nil {
		return nil, err
	}
	return qnet.Listen(ql), nil
}

func NewServer(addr string, opts ...options.ServerOption) (*grpc.Server, net.Listener, error) {
	cfg := options.NewServerConfig()
	if err := cfg.Apply(opts...); err != nil {
		return nil, nil, err
	}
	creds := transports.NewCredentials(cfg.TLSConf)
	l, err := newListener(addr, cfg.TLSConf)
	if err != nil {
		return nil, nil, err
	}
	return grpc.NewServer(grpc.Creds(creds)), l, err
}
