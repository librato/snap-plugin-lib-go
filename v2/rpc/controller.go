package rpc

import (
	"net"
	"time"

	"github.com/librato/snap-plugin-lib-go/v2/context_manager"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const GRPCGracefulStopTimeout = 10 * time.Second

var log = logrus.WithFields(logrus.Fields{"module": "plugin-rpc"})

func StartGRPCController(proxy context_manager.Collector) {
	closeChan := make(chan error, 1)

	lis, err := net.Listen("tcp", "0.0.0.0:56789")
	if err != nil {
		log.Fatalf("can't create tcp connection (%s)", err)
	}

	grpcServer := grpc.NewServer()
	RegisterControllerServer(grpcServer, newControlService(closeChan))
	RegisterCollectorServer(grpcServer, newCollectService(proxy))

	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			closeChan <- err
		}
	}()

	<-closeChan
	shutdownPlugin(grpcServer)
}

func shutdownPlugin(grpcServer *grpc.Server) {
	stopped := make(chan bool, 1)

	// try to complete all remaining rpc calls
	go func() {
		grpcServer.GracefulStop()
		stopped <- true
	}()

	// If RPC calls lasting too much, stop server by force
	select {
	case <-stopped:
		log.Debug("GRPC server stopped gracefully")
	case <-time.After(GRPCGracefulStopTimeout):
		grpcServer.Stop()
		log.Warning("GRPC server couldn't have been stopped gracefully. Some metrics might be lost")
	}
}
