/*
 Copyright (c) 2020 SolarWinds Worldwide, LLC

    Licensed under the Apache License, Version 2.0 (the "License");
    you may not use this file except in compliance with the License.
    You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/

package proxy

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/librato/snap-plugin-lib-go/v2/internal/util/log"
	"github.com/librato/snap-plugin-lib-go/v2/internal/util/simpleconfig"
	"github.com/librato/snap-plugin-lib-go/v2/internal/util/types"
	"github.com/sirupsen/logrus"
)

const (
	maxWarningMsgSize = 256 // maximum length of a single warning message
	maxNoOfWarnings   = 40  // maximum number of warnings added during one collect/publish operation
)

var (
	moduleFields = logrus.Fields{"layer": "lib", "module": "common-proxy"}
)

type Context struct {
	rawConfig          []byte
	flattenedConfig    map[string]string
	storedObjectsMutex sync.RWMutex
	storedObjects      map[string]interface{}
	warningsMutex      sync.RWMutex
	sessionWarnings    []types.Warning

	ctx      context.Context
	cancelFn context.CancelFunc
}

type Warning struct {
	Message   string
	Timestamp time.Time
}

func NewContext(rawConfig []byte) (*Context, error) {
	flattenedConfig, err := simpleconfig.JSONToFlatMap(rawConfig)
	if err != nil {
		return nil, fmt.Errorf("can't create context due to invalid json: %v", err)
	}

	return &Context{
		rawConfig:       rawConfig,
		flattenedConfig: flattenedConfig,
		storedObjects:   map[string]interface{}{},
		ctx:             context.Background(),
	}, nil
}

func (c *Context) Config(key string) (string, bool) {
	v, ok := c.flattenedConfig[key]
	return v, ok
}

func (c *Context) ConfigKeys() []string {
	var keysList []string
	for k := range c.flattenedConfig {
		keysList = append(keysList, k)
	}
	return keysList
}

func (c *Context) RawConfig() []byte {
	return c.rawConfig
}

func (c *Context) Store(key string, obj interface{}) {
	c.storedObjectsMutex.Lock()
	defer c.storedObjectsMutex.Unlock()

	c.storedObjects[key] = obj
}

func (c *Context) Load(key string) (interface{}, bool) {
	c.storedObjectsMutex.RLock()
	defer c.storedObjectsMutex.RUnlock()

	obj, ok := c.storedObjects[key]
	return obj, ok
}

func (c *Context) LoadTo(key string, dest interface{}) error {
	c.storedObjectsMutex.RLock()
	defer c.storedObjectsMutex.RUnlock()

	obj, ok := c.storedObjects[key]
	if !ok {
		return fmt.Errorf("couldn't find object with a given key (%s)", key)
	}

	vDest := reflect.ValueOf(dest)
	if vDest.Kind() != reflect.Ptr || vDest.IsNil() {
		return fmt.Errorf("passed variable should be a non-nill pointer")
	}
	if reflect.TypeOf(dest).Elem() != reflect.TypeOf(obj) {
		return fmt.Errorf("type of destination variable don't match to type of stored value")
	}

	vDest.Elem().Set(reflect.ValueOf(obj))

	return nil
}

func (c *Context) AddWarning(msg string) {
	logF := log.WithCtx(c.ctx).WithFields(moduleFields).WithField("service", "proxy")

	if c.IsDone() {
		logF.Warn("task has been canceled")
		return
	}

	c.warningsMutex.Lock()
	defer c.warningsMutex.Unlock()

	if len(c.sessionWarnings) >= maxNoOfWarnings {
		logF.Warn("Maximum number of warnings logged. New warning has been ignored")
		return
	}

	if len(msg) > maxWarningMsgSize {
		logF.Info("Warning message size exceeds maximum allowed value and will be cut off")
		msg = msg[:maxWarningMsgSize]
	}

	c.sessionWarnings = append(c.sessionWarnings, types.Warning{
		Message:   msg,
		Timestamp: time.Now(),
	})
}

func (c *Context) Warnings(clear bool) []types.Warning {
	c.warningsMutex.RLock()
	defer c.warningsMutex.RUnlock()

	warnings := c.sessionWarnings
	if clear {
		warnings = []types.Warning{}
	}
	return warnings
}

func (c *Context) ResetWarnings() {
	c.warningsMutex.RLock()
	defer c.warningsMutex.RUnlock()

	c.sessionWarnings = []types.Warning{}
}

func (c *Context) IsDone() bool {
	return c.ctx.Err() != nil
}

func (c *Context) Done() <-chan struct{} {
	return c.ctx.Done()
}

func (c *Context) AttachContext(parentCtx context.Context) {
	c.ctx, c.cancelFn = context.WithCancel(parentCtx)
}

func (c *Context) ReleaseContext() {
	c.cancelFn()
}
