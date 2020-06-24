/*
The package "plugin" provides interfaces to define custom plugins and Context interface
which allows to perform any collection-related operation.
*/
package plugin

type Collector interface {
	Collect(ctx CollectContext) error
}

type StreamingCollector interface {
	StreamingCollect(ctx CollectContext) error
}

type LoadableCollector interface {
	Load(ctx Context) error
}

type UnloadableCollector interface {
	Unload(ctx Context) error
}

type DefinableCollector interface {
	PluginDefinition(def CollectorDefinition) error
}

type CustomizableInfoCollector interface {
	CustomInfo(ctx Context) interface{}
}

///////////////////////////////////////////////////////////////////////////////

// CollectContext provides metric, state and configuration API to be used by custom code.
type CollectContext interface {
	Context

	// Add concrete metric with calculated value
	AddMetric(namespace string, value interface{}, modifier ...MetricModifier) error

	// Always apply specific modifier(s) for a metrics matching namespace selector
	// Returns object which may be used to saturate modifiers (make them no-active)
	AlwaysApply(namespaceSelector string, modifier ...MetricModifier) (Saturator, error)

	// Provide information whether metric or metric group is reasonable to process (won't be filtered).
	ShouldProcess(namespace string) bool

	// List all requested metrics (filter).
	// WARNING: library automatically filters metrics based on provided list. You should use this function
	// in scenarios when output metrics namespaces are constructed based on input list (ie. snmp metrics based on OIDs)
	RequestedMetrics() []string
}

///////////////////////////////////////////////////////////////////////////////

// CollectorDefinition provides API for specifying plugin metadata (supported metrics, descriptions etc)
type CollectorDefinition interface {
	Definition

	// Define supported metric, its description and indication if metric is default
	DefineMetric(namespace string, unit string, isDefault bool, description string)

	// Define description for dynamic element
	DefineGroup(name string, description string)

	// Define global tags that will be applied to all metrics
	DefineGlobalTags(namespaceSelector string, tags map[string]string)

	// Define example config (which will be presented when example task is printed)
	DefineExampleConfig(cfg string) error
}
