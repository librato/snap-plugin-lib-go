package collector

import (
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	proxy2 "github.com/librato/snap-plugin-lib-go/v2/tutorial/07-proxy/collector/proxy"
)

type systemCollector struct {
	proxyCollector proxy2.Proxy
}

func (s systemCollector) PluginDefinition(def plugin.CollectorDefinition) error {
	return nil
}

func New(proxy proxy2.Proxy) systemCollector {
	return systemCollector{
		proxyCollector: proxy,
	}
}

func (s systemCollector) Collect(plugin.CollectContext) error {
	return nil
}