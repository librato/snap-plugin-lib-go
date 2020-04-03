/*
Package rpc:
* contains Protocol Buffer types definitions
* handles GRPC communication (server side), passing it to proxies.
* contains Implementation of GRPC services.
*/
package service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"time"

	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	"google.golang.org/grpc/credentials"

	"github.com/fullstorydev/grpchan"
	"github.com/librato/snap-plugin-lib-go/v2/pluginrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const GRPCGracefulStopTimeout = 10 * time.Second

var log = logrus.WithFields(logrus.Fields{"layer": "lib", "module": "plugin-rpc"})

type Server interface {
	grpchan.ServiceRegistry

	// For compatibility with the native grpc.Server
	Serve(lis net.Listener) error
	GracefulStop()
	Stop()
}

// An abstraction providing a unified interface for
// * the native go-grpc implementation
// * https://github.com/fullstorydev/grpchan - this one provides a way of using gRPC with a custom transport
//   (that means sth other than the native h2 - HTTP1.1 or inprocess/channels are available out of the box)
func NewGRPCServer(opt *plugin.Options) (Server, error) {
	if opt.AsThread {
		return NewChannel(), nil
	}

	if !opt.EnableTLS {
		return grpc.NewServer(), nil
	}

	cert, err := tls.LoadX509KeyPair(opt.TLSServerCertPath, opt.TLSServerKeyPath)
	if err != nil {
		return nil, fmt.Errorf("invalid TLS certificate: %v", err)
	}

	clientCA := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(opt.TLSClientCARootPath)
	if err != nil {
		return nil, fmt.Errorf("can't read client CA Root certificate: %v", err)
	}
	clientCA.AppendCertsFromPEM(caCert)

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    clientCA,
	}
	tlsCreds := credentials.NewTLS(tlsConfig)

	return grpc.NewServer(grpc.Creds(tlsCreds)), nil
}

func StartCollectorGRPC(srv Server, proxy CollectorProxy, grpcLn net.Listener, pingTimeout time.Duration, pingMaxMissedCount uint) {
	pluginrpc.RegisterHandlerCollector(srv, newCollectService(proxy))
	startGRPC(srv, grpcLn, pingTimeout, pingMaxMissedCount)
}

func StartPublisherGRPC(srv Server, proxy PublisherProxy, grpcLn net.Listener, pingTimeout time.Duration, pingMaxMissedCount uint) {
	pluginrpc.RegisterHandlerPublisher(srv, newPublishingService(proxy))
	startGRPC(srv, grpcLn, pingTimeout, pingMaxMissedCount)
}

func startGRPC(srv Server, grpcLn net.Listener, pingTimeout time.Duration, pingMaxMissedCount uint) {
	closeChan := make(chan error, 1)
	pluginrpc.RegisterHandlerController(srv, newControlService(closeChan, pingTimeout, pingMaxMissedCount))

	go func() {
		err := srv.Serve(grpcLn) // may be blocking (depending on implementation)
		if err != nil {
			closeChan <- err
		}
	}()

	exitErr := <-closeChan // may be blocking (depending on implementation)

	if exitErr != nil && exitErr != RequestedKillError {
		log.WithError(exitErr).Errorf("Major error occurred - plugin will be shut down")
	}

	shutdownPlugin(srv)
}

func shutdownPlugin(srv Server) {
	stopped := make(chan bool, 1)

	// try to complete all remaining rpc calls
	go func() {
		srv.GracefulStop()
		stopped <- true
	}()

	// If RPC calls lasting too much, stop server by force
	select {
	case <-stopped:
		log.Debug("GRPC server stopped gracefully")
	case <-time.After(GRPCGracefulStopTimeout):
		srv.Stop()
		log.Warning("GRPC server couldn't have been stopped gracefully. Some metrics might have been lost")
	}
}
