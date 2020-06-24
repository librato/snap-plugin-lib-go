package runner

import (
	"fmt"
	"os"

	"github.com/librato/snap-plugin-lib-go/v2/internal/plugins/common/stats"
	"github.com/librato/snap-plugin-lib-go/v2/internal/plugins/publisher/proxy"
	"github.com/librato/snap-plugin-lib-go/v2/internal/service"
	"github.com/librato/snap-plugin-lib-go/v2/internal/util/types"
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	"github.com/sirupsen/logrus"
)

func StartPublisher(publisher plugin.Publisher, name string, version string) {
	var err error

	var opt *plugin.Options
	inprocPlugin, inProc := publisher.(inProcessPlugin)
	if inProc {
		opt = inprocPlugin.Options()
	}

	if opt == nil {
		opt, err = ParseCmdLineOptions(os.Args[0], types.PluginTypePublisher, os.Args[1:])
		if err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "Error occured during plugin startup (%v)\n", err)
			os.Exit(errorExitStatus)
		}
	}

	err = ValidateOptions(opt)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Invalid plugin options (%v)\n", err)
		os.Exit(errorExitStatus)
	}

	statsController, err := stats.NewController(name, version, types.PluginTypePublisher, opt)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error occured when starting statistics controller (%v)\n", err)
		os.Exit(errorExitStatus)
	}

	ctxMan := proxy.NewContextManager(publisher, statsController)

	logrus.SetLevel(opt.LogLevel)

	if opt.PrintVersion {
		printVersion(name, version)
		os.Exit(normalExitStatus)
	}

	r, err := acquireResources(opt)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Can't acquire resources for plugin services (%v)\n", err)
		os.Exit(errorExitStatus)
	}

	jsonMeta := metaInformation(name, version, types.PluginTypePublisher, opt, r, ctxMan.TasksLimit, ctxMan.InstancesLimit)
	if inProc {
		inprocPlugin.MetaChannel() <- jsonMeta
		close(inprocPlugin.MetaChannel())
	}

	if opt.EnableProfiling {
		startPprofServer(r.pprofListener)
		defer r.pprofListener.Close() // close pprof service when GRPC service has been shut down
	}

	if opt.EnableStatsServer {
		startStatsServer(r.statsListener, statsController)
		defer r.statsListener.Close() // close stats service when GRPC service has been shut down
	}

	srv, err := service.NewGRPCServer(opt)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Can't initialize GRPC Server (%v)\n", err)
		os.Exit(errorExitStatus)
	}

	// We need to bind the gRPC client on the other end to the same channel so need to return it from here
	if inProc {
		inprocPlugin.GRPCChannel() <- srv.(*service.Channel).Channel
	}

	// main blocking operation
	service.StartPublisherGRPC(srv, ctxMan, r.grpcListener, opt.GRPCPingTimeout, opt.GRPCPingMaxMissed)
}
