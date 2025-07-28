package weave

import "sync"

type Map[K comparable, V any] struct {
	mu   sync.RWMutex
	data map[K]V
}

func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{
		data: make(map[K]V),
	}
}

func (m *Map[K, V]) Set(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data[key] = value
}

func (m *Map[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	value, ok := m.data[key]
	return value, ok
}

func (m *Map[K, V]) Delete(key K) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.data, key)
}

func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	for key, value := range m.data {
		if !f(key, value) {
			break
		}
	}
}

func (m *Map[K, V]) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

func (m *Map[K, V]) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[K]V)
}

func (m *Map[K, V]) Keys() []K {
	m.mu.RLock()
	defer m.mu.RUnlock()
	keys := make([]K, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys
}

func (m *Map[K, V]) Values() []V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	values := make([]V, 0, len(m.data))
	for _, value := range m.data {
		values = append(values, value)
	}
	return values
}

func (m *Map[K, V]) Contains(key K) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.data[key]
	return ok
}

func (m *Map[K, V]) IsEmpty() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data) == 0
}

func (m *Map[K, V]) ToMap() map[K]V {
	m.mu.RLock()
	defer m.mu.RUnlock()
	copied := make(map[K]V, len(m.data))
	for k, v := range m.data {
		copied[k] = v
	}
	return copied
}
