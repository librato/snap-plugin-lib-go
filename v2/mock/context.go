package mock

import (
	"github.com/librato/snap-plugin-lib-go/v2/plugin"
	"github.com/stretchr/testify/mock"
)

type Context struct {
	mock.Mock
}

func (m *Context) Config(key string) (string, bool) {
	args := m.Called(key)
	return args.String(0), args.Bool(1)
}

func (m *Context) ConfigKeys() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *Context) RawConfig() []byte {
	args := m.Called()
	return args.Get(0).([]byte)
}

func (m *Context) Store(key string, value interface{}) {
	m.Called(key, value)
}

func (m *Context) Load(key string) (interface{}, bool) {
	args := m.Called(key)
	return args.Get(0), args.Bool(1)
}

func (m *Context) LoadTo(key string, dest interface{}) error {
	args := m.Called(key, dest)
	return args.Error(0)
}

func (m *Context) AddWarning(msg string) {
	m.Called(msg)
}

func (m *Context) AddMetric(ns string, value interface{}, modifiers ...plugin.MetricModifier) error {
	args := m.Called(ns, value, modifiers)
	return args.Error(0)
}

func (m *Context) AlwaysApply(namespaceSelector string, modifiers ...plugin.MetricModifier) error {
	args := m.Called(namespaceSelector, modifiers)
	return args.Error(0)
}

func (m *Context) ShouldProcess(ns string) bool {
	args := m.Called(ns)
	return args.Bool(0)
}

func (m *Context) RequestedMetrics() []string {
	args := m.Called()
	return args.Get(0).([]string)
}

func (m *Context) IsDone() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *Context) Done() <-chan struct{} {
	args := m.Called()
	return args.Get(0).(<-chan struct{})
}

// publisher context
func (m *Context) ListAllMetrics() []plugin.Metric {
	args := m.Called()
	return args.Get(0).([]plugin.Metric)
}

func (m *Context) Count() int {
	args := m.Called()
	return args.Int(0)
}
