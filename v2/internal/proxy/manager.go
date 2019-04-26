/*
Package proxy:
1) Manages context for different task created for the same plugin
2) Serves as an entry point for any "controller" (like. Rpc)
*/
package proxy

import (
	"errors"
	"fmt"
	"sync"

	"github.com/librato/snap-plugin-lib-go/v2/internal/util/metrictree"
	"github.com/librato/snap-plugin-lib-go/v2/internal/util/types"
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	"github.com/sirupsen/logrus"
)

var log *logrus.Entry

func init() {
	log = logrus.WithFields(logrus.Fields{"layer": "lib", "module": "plugin-proxy"})
}

type Collector interface {
	RequestCollect(id int) ([]*types.Metric, error)
	LoadTask(id int, config []byte, selectors []string) error
	UnloadTask(id int) error
	RequestInfo()
}

type metricMetadata struct {
	isDefault   bool
	description string
	unit        string
}

type ContextManager struct {
	collector  plugin.Collector       // reference to custom plugin code
	contextMap map[int]*pluginContext // map of contexts associated with taskIDs

	activeTasks      map[int]struct{} // map of active tasks (tasks for which Collect RPC request is progressing)
	activeTasksMutex sync.RWMutex     // mutex associated with activeTasks

	metricsDefinition *metrictree.TreeValidator // metrics defined by plugin (code)

	metricsMetadata   map[string]metricMetadata // metadata associated with each metric (is default?, description, unit)
	groupsDescription map[string]string         // description associated with each group (dynamic element)
}

func NewContextManager(collector plugin.Collector, _, _ string) *ContextManager {
	cm := &ContextManager{
		collector:   collector,
		contextMap:  map[int]*pluginContext{},
		activeTasks: map[int]struct{}{},

		metricsDefinition: metrictree.NewMetricDefinition(),

		metricsMetadata:   map[string]metricMetadata{},
		groupsDescription: map[string]string{},
	}

	cm.RequestPluginDefinition()

	return cm
}

///////////////////////////////////////////////////////////////////////////////
// proxy.Collector related methods

func (cm *ContextManager) RequestCollect(id int) ([]*types.Metric, error) {
	if cm.tryToActivateTask(id) {
		return nil, fmt.Errorf("can't process collect request, other request for the same id (%d) is in progress", id)
	}
	defer cm.markTaskAsCompleted(id)

	context, ok := cm.contextMap[id]
	if !ok {
		return nil, fmt.Errorf("can't find a context for a given id: %d", id)
	}

	// collect metrics - user defined code
	context.sessionMts = []*types.Metric{}
	err := cm.collector.Collect(context)
	if err != nil {
		return nil, fmt.Errorf("user-defined Collect method ended with error: %v", err)
	}

	return context.sessionMts, nil
}

func (cm *ContextManager) LoadTask(id int, rawConfig []byte, mtsFilter []string) error {
	if cm.tryToActivateTask(id) {
		return fmt.Errorf("can't process load request, other request for the same id (%d) is in progress", id)
	}
	defer cm.markTaskAsCompleted(id)

	if _, ok := cm.contextMap[id]; ok {
		return errors.New("context with given id was already defined")
	}

	newCtx, err := NewPluginContext(cm, rawConfig)
	if err != nil {
		return fmt.Errorf("can't load task: %v", err)
	}

	for _, mtFilter := range mtsFilter {
		err := newCtx.metricsFilters.AddRule(mtFilter)
		if err != nil {
			log.WithError(err).WithField("rule", mtFilter).Warn("can't add filtering rule, it will be ignored")
		}
	}

	if loadable, ok := cm.collector.(plugin.LoadableCollector); ok {
		err := loadable.Load(newCtx)
		if err != nil {
			return fmt.Errorf("can't load task due to errors returned from user-defined function: %s", err)
		}
	}

	cm.contextMap[id] = newCtx

	return nil
}

func (cm *ContextManager) UnloadTask(id int) error {
	if cm.tryToActivateTask(id) {
		return fmt.Errorf("can't process unload request, other request for the same id (%d) is in progress", id)
	}
	defer cm.markTaskAsCompleted(id)

	if _, ok := cm.contextMap[id]; !ok {
		return errors.New("context with given id is not defined")
	}

	if loadable, ok := cm.collector.(plugin.LoadableCollector); ok {
		err := loadable.Unload(cm.contextMap[id])
		if err != nil {
			return fmt.Errorf("error occured when trying to unload a task (%d): %v", id, err)
		}
	}

	delete(cm.contextMap, id)
	return nil
}

func (cm *ContextManager) RequestInfo() {
	return
}

///////////////////////////////////////////////////////////////////////////////
// plugin.CollectorDefinition related methods

func (cm *ContextManager) DefineMetric(ns string, unit string, isDefault bool, description string) {
	err := cm.metricsDefinition.AddRule(ns)
	if err != nil {
		log.WithError(err).WithFields(logrus.Fields{"namespace": ns}).Errorf("Wrong metric definition")
	}

	cm.metricsMetadata[ns] = metricMetadata{
		isDefault:   isDefault,
		description: description,
		unit:        unit,
	}
}

// Define description for dynamic element
func (cm *ContextManager) DefineGroup(name string, description string) {
	cm.groupsDescription[name] = description
}

// Define global tags that will be applied to all metrics
func (cm *ContextManager) DefineGlobalTags(string, map[string]string) {
	panic("implement")
}

///////////////////////////////////////////////////////////////////////////////

func (cm *ContextManager) RequestPluginDefinition() {
	if definable, ok := cm.collector.(plugin.DefinableCollector); ok {
		err := definable.DefineMetrics(cm)
		if err != nil {
			log.WithError(err).Errorf("Error occurred during plugin definition")
		}
	}
}

func (cm *ContextManager) tryToActivateTask(id int) bool {
	cm.activeTasksMutex.Lock()
	defer cm.activeTasksMutex.Unlock()

	if _, ok := cm.activeTasks[id]; ok {
		return true
	}

	cm.activeTasks[id] = struct{}{}
	return false
}

func (cm *ContextManager) markTaskAsCompleted(id int) {
	cm.activeTasksMutex.Lock()
	defer cm.activeTasksMutex.Unlock()

	delete(cm.activeTasks, id)
}
