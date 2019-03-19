package runner

import (
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	"github.com/librato/snap-plugin-lib-go/v2/proxy"
	"github.com/librato/snap-plugin-lib-go/v2/rpc"
)

func StartCollector(collector plugin.Collector, name string, version string) {

	contextManager := proxy.NewContextManager(collector, name, version)
	rpc.StartGRPCController(contextManager)
}