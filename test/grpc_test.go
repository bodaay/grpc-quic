package test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	qgrpc "github.com/bodaay/grpc-quic"
	"github.com/bodaay/grpc-quic/opts"
	"github.com/bodaay/grpc-quic/proto/hello"
	"google.golang.org/grpc"
	"google.golang.org/grpc/resolver/manual"
)

type Hello struct{}

func (h *Hello) SayHello(ctx context.Context, in *hello.HelloRequest) (*hello.HelloReply, error) {
	rep := new(hello.HelloReply)
	rep.Message = "Hello " + in.GetName()
	return rep, nil
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig() (*tls.Config, error) {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}

	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		return nil, err
	}

	return &tls.Config{Certificates: []tls.Certificate{tlsCert}, NextProtos: []string{"http/2"}}, nil
}

func testDial(t *testing.T, target string) {
	var (
		client *grpc.ClientConn
		server *grpc.Server

		err error
	)

	defer func() {
		if client != nil {
			client.Close()
		}

		if server != nil {
			server.Stop()
		}
	}()

	t.Run("Setup server", func(t *testing.T) {
		//setup server
		tlsConf, err := generateTLSConfig()
		if err != nil {
			t.Fail()
		}

		server, l, err := qgrpc.NewServer(target, opts.TLSConfig(tlsConf))
		if err != nil {
			t.Fail()
		}

		hello.RegisterGreeterServer(server, &Hello{})

		go func() {
			err := server.Serve(l)
			if err != nil {
				t.Fail()
			}
		}()
	})

	t.Run("Setup client", func(t *testing.T) {
		tlsConf := &tls.Config{InsecureSkipVerify: true, NextProtos: []string{"http/2"}}

		// Take a random port to listen from udp server
		client, err = qgrpc.Dial(target, opts.WithTLSConfig(tlsConf))
		if err != nil {
			t.Fail()
		}
	})

	t.Run("Test basic dial", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		greet := hello.NewGreeterClient(client)
		req := new(hello.HelloRequest)
		req.Name = "World"

		rep, err := greet.SayHello(ctx, req)
		if err != nil {
			t.Fatal(err)
		}
		if rep.GetMessage() != "Hello World" {
			t.Fatal("message not equal")
		}
	})
}

func TestDialUDP(t *testing.T) {
	target := "127.0.0.1:5847"
	testDial(t, target)
}

type testHandler func(*manual.Resolver, hello.GreeterClient, []*grpc.Server)
