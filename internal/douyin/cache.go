package douyin

import (
	"sync"
	"time"
)

const DefaultTTL = 3600 * time.Second

type entry struct {
	data  interface{}
	expAt time.Time
}

type TTL struct {
	mu    sync.RWMutex
	items map[string]*entry
	ttl   time.Duration
}

func NewTTL(ttl time.Duration) *TTL {
	if ttl <= 0 {
		ttl = DefaultTTL
	}
	return &TTL{
		items: make(map[string]*entry),
		ttl:   ttl,
	}
}

func (c *TTL) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, ok := c.items[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(e.expAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}
	return e.data, true
}

func (c *TTL) Set(key string, data interface{}) {
	c.mu.Lock()
	c.items[key] = &entry{
		data:  data,
		expAt: time.Now().Add(c.ttl),
	}
	c.mu.Unlock()
}

func (c *TTL) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}
