// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"sync"
	"time"

	"github.com/benbjohnson/clock"
)

type entryType interface {
	any
}

type entry[E entryType] struct {
	expiry *time.Time
	value  *E
}

type TTLMap[E entryType] struct {
	data map[string]entry[E]
	clock.Clock
	sync.Mutex
}

func newTTLMap[E entryType]() *TTLMap[E] {
	return &TTLMap[E]{
		data:  map[string]entry[E]{},
		Clock: clock.New(),
		Mutex: sync.Mutex{},
	}
}

func (m *TTLMap[E]) Get(key string) (*E, bool) {
	m.Lock()
	defer m.Unlock()

	if entry, found := m.data[key]; found {
		return entry.value, true
	}
	return nil, false
}

func (m *TTLMap[E]) Size() int {
	m.Lock()
	defer m.Unlock()

	return len(m.data)
}

func (m *TTLMap[E]) Put(key string, value *E) {
	m.Lock()
	defer m.Unlock()

	m.data[key] = entry[E]{value: value, expiry: nil}
}

func (m *TTLMap[E]) SetTTL(key string, ttl time.Duration) {
	m.Lock()
	defer m.Unlock()

	if entry, found := m.data[key]; found {
		expiry := m.Now().Add(ttl)
		entry.expiry = &expiry
		m.data[key] = entry
	}
}

func (m *TTLMap[E]) MapCopy() map[string]E {
	m.Lock()
	defer m.Unlock()

	mapCopy := make(map[string]E, len(m.data))
	for key, entry := range m.data {
		mapCopy[key] = *entry.value
	}
	return mapCopy
}

func (m *TTLMap[E]) Sweep(currentTime time.Time) {
	m.Lock()
	defer m.Unlock()

	for key, entry := range m.data {
		if entry.expiry == nil {
			continue
		}
		if entry.expiry.Before(currentTime) {
			delete(m.data, key)
		}
	}
}
