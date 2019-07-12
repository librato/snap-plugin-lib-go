package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/librato/snap-plugin-lib-go/v2/internal/pluginrpc"
	"google.golang.org/grpc"
)

const (
	defaultGRPCIP          = "127.0.0.1"
	defaultGRPCPort        = 0
	defaultConfig          = "{}"
	defaultFilter          = ""
	defaultTaskID          = 0
	defaultCollectInterval = 5 * time.Second
	defaultPingInterval    = 2 * time.Second

	grpcLoadDelay      = 500 * time.Millisecond
	grpcRequestTimeout = 10 * time.Second

	filterSeparator = ";"

	maxTaskId = 1024
)

type Options struct {
	PluginIP           string
	PluginPort         int
	CollectInterval    time.Duration
	PingInterval       time.Duration
	MaxCollectRequests int
	SendKill           bool

	PluginConfig string
	PluginFilter string
	TaskId       int
}

const (
	unloadMaxRetry   = 5
	unloadRetryDelay = 1 * time.Second

	stoppedByUser = 1
)

///////////////////////////////////////////////////////////////////////////////

func parseCmdLine() *Options {
	opt := &Options{}

	flag.StringVar(&opt.PluginIP,
		"plugin-ip", defaultGRPCIP,
		"IP Address of GRPC Server run by plugin")

	flag.IntVar(&opt.PluginPort,
		"plugin-port", defaultGRPCPort,
		"Port of GRPC Server run by plugin")

	flag.IntVar(&opt.TaskId,
		"task-id", defaultTaskID,
		"Task identifier used to make GRPC requests (0 means random)")

	flag.StringVar(&opt.PluginConfig,
		"plugin-config", defaultConfig,
		"Plugin configuration (should be valid JSON)")

	flag.StringVar(&opt.PluginFilter,
		"plugin-filter", defaultFilter,
		"Plugin filter (definition which subset of metrics should be gathered), ie. '/example/static/*;/example/global/*'")

	flag.IntVar(&opt.MaxCollectRequests,
		"max-collect-requests", 0,
		"Maximum number of collect requests (default 0 for infinite)")

	flag.DurationVar(&opt.CollectInterval,
		"collect-interval", defaultCollectInterval,
		"Duration between Collect requests")

	flag.DurationVar(&opt.PingInterval,
		"ping-interval", defaultPingInterval,
		"Duration between Ping requests")

	flag.BoolVar(&opt.SendKill,
		"send-kill", false,
		"When set, Kill request will be set after 'max-collect-requests' collects ")

	flag.Parse()

	if opt.TaskId == defaultTaskID {
		opt.TaskId = rand.Intn(maxTaskId)
	}

	return opt
}

///////////////////////////////////////////////////////////////////////////////

func main() {
	rand.Seed(time.Now().UnixNano())

	doneCh := make(chan error)

	opt := parseCmdLine()

	// Create connection
	grpcServerAddr := fmt.Sprintf("%s:%d", opt.PluginIP, opt.PluginPort)
	cl, err := grpc.Dial(grpcServerAddr, grpc.WithInsecure())
	if err != nil {
		fmt.Printf("Can't to GRPC Server on %s", grpcServerAddr)
		os.Exit(1)
	}
	defer func() { _ = cl.Close() }()

	// Load, collect, unload routine
	go func() {
		collClient := pluginrpc.NewCollectorClient(cl)

		err := doLoadRequest(collClient, opt)
		if err != nil {
			doneCh <- fmt.Errorf("can't send load request to plugin: %v", err)
		}

		// Handle ctrl+C
		notifyCh := make(chan os.Signal, 1)
		signal.Notify(notifyCh, os.Interrupt)
		go func() {
			<-notifyCh
			fmt.Printf("!! Ctrl+c pressed !! trying to unload current task\n")

			for i := 0; i < unloadMaxRetry; i++ {
				err := doUnloadRequest(collClient, opt)
				if err != nil {
					fmt.Printf("!! Can't unload plugin (%v), will retry (%d/%d)...\n", err, i+1, unloadMaxRetry)
					time.Sleep(unloadRetryDelay)
					continue
				}

				break
			}

			os.Exit(stoppedByUser)
		}()

		time.Sleep(grpcLoadDelay)

		reqCounter := 0
		for {
			reqCounter++
			recvMts, err := doCollectRequest(collClient, opt)
			if err != nil {
				doneCh <- fmt.Errorf("can't send collect request to plugin: %v", err)
			}

			fmt.Printf("Recived %d metric(s)\n", len(recvMts))
			for _, mt := range recvMts {
				fmt.Printf(" %s\n", mt)
			}
			fmt.Printf("\n")

			if reqCounter == opt.MaxCollectRequests {
				break
			}
			time.Sleep(opt.CollectInterval)
		}

		time.Sleep(grpcLoadDelay)

		err = doUnloadRequest(collClient, opt)
		if err != nil {
			doneCh <- fmt.Errorf("can't send unload request to plugin: %v", err)
		}

		doneCh <- nil
	}()

	// ping routine
	contClient := pluginrpc.NewControllerClient(cl)

	go func() {
		for {
			req := &pluginrpc.PingRequest{}
			_, err := contClient.Ping(context.Background(), req)
			if err != nil {
				doneCh <- fmt.Errorf("can't start ")
			}
			time.Sleep(opt.PingInterval)
		}
	}()

	doneErr := <-doneCh

	if opt.SendKill {
		err := doKillRequest(contClient)
		if err != nil {
			doneCh <- fmt.Errorf("can't send kill request to plugin: %v", err)
		}
	}

	if doneErr != nil {
		fmt.Printf("Snap-mock exists because of error: %v", doneErr)
	}
}

///////////////////////////////////////////////////////////////////////////////

func doLoadRequest(cc pluginrpc.CollectorClient, opt *Options) error {
	filter := []string{}
	if opt.PluginFilter != defaultFilter {
		filter = strings.Split(opt.PluginFilter, filterSeparator)
	}

	reqLoad := &pluginrpc.LoadRequest{
		TaskId:          int32(opt.TaskId),
		JsonConfig:      []byte(opt.PluginConfig),
		MetricSelectors: filter,
	}

	ctx, fn := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer fn()

	_, err := cc.Load(ctx, reqLoad)

	return err
}

func doUnloadRequest(cc pluginrpc.CollectorClient, opt *Options) error {
	reqUnload := &pluginrpc.UnloadRequest{
		TaskId: int32(opt.TaskId),
	}

	ctx, fn := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer fn()

	_, err := cc.Unload(ctx, reqUnload)

	return err
}

func doKillRequest(cc pluginrpc.ControllerClient) error {
	reqKill := &pluginrpc.KillRequest{}

	ctx, fn := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer fn()

	_, err := cc.Kill(ctx, reqKill)
	return err
}

func doCollectRequest(cc pluginrpc.CollectorClient, opt *Options) ([]string, error) {
	var recvMts []string

	reqColl := &pluginrpc.CollectRequest{
		TaskId: int32(opt.TaskId),
	}

	ctx, fn := context.WithTimeout(context.Background(), grpcRequestTimeout)
	defer fn()

	stream, err := cc.Collect(ctx, reqColl)
	if err != nil {
		return recvMts, fmt.Errorf("can't send collect request to plugin: %v", err)
	}

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return recvMts, fmt.Errorf("error when receiving collect reply from plugin (%v)", err)
		}

		for _, mt := range resp.MetricSet {
			recvMts = append(recvMts, grpcMetricToString(mt))
		}
	}

	return recvMts, nil
}

///////////////////////////////////////////////////////////////////////////////

func grpcMetricToString(metric *pluginrpc.Metric) string {
	var nsStr []string
	for _, ns := range metric.Namespace {
		nsStr = append(nsStr, ns.Value)
	}

	return fmt.Sprintf("%s %v [%v]", strings.Join(nsStr, "."), metric.Value, metric.Tags)
}
