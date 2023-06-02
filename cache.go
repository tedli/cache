package cache

import (
	"context"
	"sync"
	"time"
)

func WithTTL(d time.Duration) func(c *config) {
	return func(c *config) {
		c.ttl = d
	}
}

func WithRefreshInterval(d time.Duration) func(c *config) {
	return func(c *config) {
		c.refreshInterval = d
	}
}

func WithGetter[Key comparable, Value any](getter func(context.Context, Key) (Value, error)) func(c *Cache[Key, Value]) {
	return func(c *Cache[Key, Value]) {
		c.getter = getter
	}
}

func WithOnRefreshError[Key comparable, Value any](onRefreshError func(Key, Value, error, func())) func(c *Cache[Key, Value]) {
	return func(c *Cache[Key, Value]) {
		c.onRefreshError = onRefreshError
	}
}

func WithBehaviour(behaviour RefreshBehaviour) func(c *config) {
	return func(c *config) {
		c.behaviour = behaviour
	}
}

func BuildCache[Key comparable, Value any](options ...func(*Cache[Key, Value])) func(...func(*config)) *Cache[Key, Value] {
	c := &Cache[Key, Value]{
		holder: make(map[Key]*Tuple[Value, time.Time]),
		mutex:  new(sync.Mutex),
	}
	for _, option := range options {
		option(c)
	}
	return func(configs ...func(*config)) *Cache[Key, Value] {
		for _, cfg := range configs {
			cfg(&c.config)
		}
		return c
	}
}

type Tuple[A, B any] struct {
	A A
	B B
}

type RefreshBehaviour int

const (
	RefreshBehaviourUndefined     = 0
	RefreshBehaviourRemoveOnly    = 1
	RefreshBehaviourFetchNewValue = 2
)

type config struct {
	behaviour       RefreshBehaviour
	refreshInterval time.Duration
	ttl             time.Duration
}

type Cache[Key comparable, Value any] struct {
	config
	holder         map[Key]*Tuple[Value, time.Time]
	getter         func(context.Context, Key) (Value, error)
	onRefreshError func(Key, Value, error, func())
	mutex          *sync.Mutex
}

func (c Cache[Key, Value]) refresh(ctx context.Context) {
	if c.ttl <= 0 {
		return
	}
	c.mutex.Lock()
	keys := make([]Key, 0, len(c.holder))
	now := time.Now()
	for key, tuple := range c.holder {
		if now.Sub(tuple.B) > c.ttl {
			keys = append(keys, key)
		}
	}
	c.mutex.Unlock()
	if c.behaviour == RefreshBehaviourRemoveOnly {
		c.mutex.Lock()
		for _, key := range keys {
			delete(c.holder, key)
		}
		c.mutex.Unlock()
	} else if c.behaviour == RefreshBehaviourFetchNewValue {
		holder := make(map[Key]Value, len(keys))
		for _, key := range keys {
			if value, err := c.getter(ctx, key); err != nil {
				if c.onRefreshError != nil {
					c.onRefreshError(key, c.holder[key].A, err, func() {
						c.mutex.Lock()
						defer c.mutex.Unlock()
						delete(c.holder, key)
					})
				}
			} else {
				holder[key] = value
			}
		}
		c.mutex.Lock()
		for key, value := range holder {
			if tuple, exist := c.holder[key]; exist {
				tuple.A = value
				tuple.B = now
			} else {
				c.holder[key] = &Tuple[Value, time.Time]{A: value, B: now}
			}
		}
		c.mutex.Unlock()
	}
}

func (c Cache[Key, Value]) Start(ctx context.Context) (_ error) {
	if c.refreshInterval <= 0 {
		<-ctx.Done()
		return
	}
	ticker := time.NewTicker(c.refreshInterval)
	for keepWorking := true; keepWorking; {
		select {
		case <-ctx.Done():
			keepWorking = false
			ticker.Stop()
		case <-ticker.C:
			c.refresh(ctx)
		}
	}
	return
}

func (c Cache[Key, Value]) Get(ctx context.Context, key Key) (value Value, err error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	tuple, cached := c.holder[key]
	if cached {
		value = tuple.A
		tuple.B = time.Now()
		return
	}
	if value, err = c.getter(ctx, key); err != nil {
		return
	}
	c.holder[key] = &Tuple[Value, time.Time]{A: value, B: time.Now()}
	return
}
