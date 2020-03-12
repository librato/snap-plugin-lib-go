package types

import (
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
)

// Simple wrapper which enables using common code for collector and streaming collector.
type Collector interface {
	plugin.Collector
	plugin.StreamingCollector

	Name() string
	Version() string
	Type() PluginType

	Unwrap() interface{} // Returns wrapped user-defined collector or streaming collector (with access to Load(), Unload() etc.)
}

func NewCollector(name string, version string, collector plugin.Collector) Collector {
	return &collectorWrapper{
		collector: collector,
		name:      name,
		version:   version,
		typ:       PluginTypeCollector,
	}
}

func NewStreamingCollector(name string, version string, collector plugin.StreamingCollector) Collector {
	return &collectorWrapper{
		streamingCollector: collector,
		name:               name,
		version:            version,
		typ:                PluginTypeStreamingCollector,
	}
}

type collectorWrapper struct {
	collector          plugin.Collector
	streamingCollector plugin.StreamingCollector

	name    string
	version string
	typ     PluginType
}

func (c *collectorWrapper) Collect(ctx plugin.CollectContext) error {
	return c.collector.Collect(ctx)
}

func (c *collectorWrapper) StreamingCollect(ctx plugin.CollectContext) error {
	return c.streamingCollector.StreamingCollect(ctx)
}

func (c *collectorWrapper) Unwrap() interface{} {
	switch c.Type() {
	case PluginTypeCollector:
		return c.collector
	case PluginTypeStreamingCollector:
		return c.streamingCollector
	default:
		panic("invalid collector type")
	}
}

func (c *collectorWrapper) Type() PluginType {
	return c.typ
}

func (c *collectorWrapper) Name() string {
	return c.name
}

func (c *collectorWrapper) Version() string {
	return c.version
}
