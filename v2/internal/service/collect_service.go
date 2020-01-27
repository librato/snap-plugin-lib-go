package service

import (
	"context"
	"fmt"
	"net"

	"github.com/librato/snap-plugin-lib-go/v2/internal/plugins/common/stats"
	"github.com/librato/snap-plugin-lib-go/v2/pluginrpc"
)

const (
	maxCollectChunkSize = 100
)

var logCollectService = log.WithField("service", "Collect")

type collectService struct {
	proxy           CollectorProxy
	statsController stats.Controller
	pprofLn         net.Listener
}

func newCollectService(proxy CollectorProxy, statsController stats.Controller, pprofLn net.Listener) pluginrpc.CollectorServer {
	return &collectService{
		proxy:           proxy,
		statsController: statsController,
		pprofLn:         pprofLn,
	}
}

func (cs *collectService) Collect(request *pluginrpc.CollectRequest, stream pluginrpc.Collector_CollectServer) error {
	logCollectService.Debug("GRPC Collect() received")

	taskID := request.GetTaskId()

	pluginMts, err := cs.proxy.RequestCollect(taskID)
	if err.Error != nil {
		return fmt.Errorf("plugin is not able to collect metrics: %s", err)
	}

	protoMts := make([]*pluginrpc.Metric, 0, len(pluginMts))
	for i, pluginMt := range pluginMts {
		protoMt, err := toGRPCMetric(pluginMt)
		if err != nil {
			logCollectService.WithError(err).WithField("metric", pluginMt.Namespace).Errorf("can't send metric over GRPC")
		}

		protoMts = append(protoMts, protoMt)

		if len(protoMts) == maxCollectChunkSize || i == len(pluginMts)-1 {
			err = stream.Send(&pluginrpc.CollectResponse{
				MetricSet: protoMts,
			})
			if err != nil {
				logCollectService.WithError(err).Error("can't send metric chunk over GRPC")
				return err
			}

			logCollectService.WithField("len", len(protoMts)).Debug("metrics chunk has been sent to snap")
			protoMts = make([]*pluginrpc.Metric, 0, len(pluginMts))
		}
	}

	return nil
}

func (cs *collectService) Load(ctx context.Context, request *pluginrpc.LoadCollectorRequest) (*pluginrpc.LoadCollectorResponse, error) {
	logCollectService.Debug("GRPC Load() received")

	taskID := string(request.GetTaskId())
	jsonConfig := request.GetJsonConfig()
	metrics := request.GetMetricSelectors()

	return &pluginrpc.LoadCollectorResponse{}, cs.proxy.LoadTask(taskID, jsonConfig, metrics)
}

func (cs *collectService) Unload(ctx context.Context, request *pluginrpc.UnloadCollectorRequest) (*pluginrpc.UnloadCollectorResponse, error) {
	logCollectService.Debug("GRPC Unload() received")

	taskID := string(request.GetTaskId())

	return &pluginrpc.UnloadCollectorResponse{}, cs.proxy.UnloadTask(taskID)
}

func (cs *collectService) Info(ctx context.Context, _ *pluginrpc.InfoRequest) (*pluginrpc.InfoResponse, error) {
	logCollectService.Debug("GRPC Info() received")

	pprofAddr := ""
	if cs.pprofLn != nil {
		pprofAddr = cs.pprofLn.Addr().String()
	}

	return serveInfo(ctx, cs.statsController.RequestStat(), pprofAddr)
}