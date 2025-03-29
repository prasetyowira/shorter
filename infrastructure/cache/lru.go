package cache

import (
	"container/list"
	"sync"
)

// NamespaceLRU is a namespace-based LRU cache implementation
type NamespaceLRU struct {
	capacity int
	items    map[string]*list.Element
	queue    *list.List
	mutex    sync.RWMutex
}

type entry struct {
	namespace string
	key       string
	value     interface{}
}

// NewNamespaceLRU creates a new namespace-based LRU cache with specified capacity
func NewNamespaceLRU(capacity int) *NamespaceLRU {
	return &NamespaceLRU{
		capacity: capacity,
		items:    make(map[string]*list.Element),
		queue:    list.New(),
	}
}

// Set adds or updates a key-value pair in the cache with a namespace
func (c *NamespaceLRU) Set(namespace, key string, value interface{}) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Create composite key for the map
	compositeKey := namespace + ":" + key

	// Check if key exists
	if element, exists := c.items[compositeKey]; exists {
		c.queue.MoveToFront(element)
		element.Value.(*entry).value = value
		return
	}

	// Add new item to the front
	element := c.queue.PushFront(&entry{
		namespace: namespace,
		key:       key,
		value:     value,
	})
	c.items[compositeKey] = element

	// Evict items if over capacity
	if c.queue.Len() > c.capacity {
		c.evict()
	}
}

// Get retrieves a value from the cache by namespace and key
func (c *NamespaceLRU) Get(namespace, key string) (interface{}, bool) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	compositeKey := namespace + ":" + key
	element, exists := c.items[compositeKey]
	if !exists {
		return nil, false
	}

	// Move to front (mark as recently used)
	c.queue.MoveToFront(element)
	return element.Value.(*entry).value, true
}

// Invalidate removes an item from the cache by namespace and key
func (c *NamespaceLRU) Invalidate(namespace, key string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	compositeKey := namespace + ":" + key
	if element, exists := c.items[compositeKey]; exists {
		c.queue.Remove(element)
		delete(c.items, compositeKey)
	}
}

// InvalidateNamespace removes all items from the specified namespace
func (c *NamespaceLRU) InvalidateNamespace(namespace string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Create a list of keys to remove to avoid modifying the map during iteration
	var keysToRemove []string
	var elementsToRemove []*list.Element

	// Identify all elements in the given namespace
	for compositeKey, element := range c.items {
		entry := element.Value.(*entry)
		if entry.namespace == namespace {
			keysToRemove = append(keysToRemove, compositeKey)
			elementsToRemove = append(elementsToRemove, element)
		}
	}

	// Remove the elements from the queue and map
	for i, key := range keysToRemove {
		c.queue.Remove(elementsToRemove[i])
		delete(c.items, key)
	}
}

// Clear empties the cache
func (c *NamespaceLRU) Clear() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.items = make(map[string]*list.Element)
	c.queue = list.New()
}

// Size returns the current number of items in the cache
func (c *NamespaceLRU) Size() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.queue.Len()
}

// evict removes the least recently used item from the cache
func (c *NamespaceLRU) evict() {
	// Get the oldest element (from the back of the queue)
	element := c.queue.Back()
	if element == nil {
		return
	}

	// Remove it from the queue
	c.queue.Remove(element)

	// Get the entry and remove it from the map
	entry := element.Value.(*entry)
	compositeKey := entry.namespace + ":" + entry.key
	delete(c.items, compositeKey)
} 