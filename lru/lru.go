package lru

import (
	"fmt"
	"reflect"
)

type LRUCache struct {
	capacity int
	cache    map[string]*Node
	head     *Node
	tail     *Node
	buffer   []string
}

type Node struct {
	key   string
	InUse int
	prev  *Node
	next  *Node
}

// Constructor creates an LRU cache with the given capacity.
func Constructor(capacity int) LRUCache {
	return LRUCache{
		capacity: capacity,
		cache:    make(map[string]*Node),
		head:     nil,
		tail:     nil,
		buffer:   make([]string, 0, capacity),
	}
}

// Get retrieves a key from the current LRU cache.
// Returns true and the value of the key if it exists.
// Returns false and "none" if it does not.
func (c *LRUCache) Get(key string, count int) (bool, string) {
	if node, ok := c.cache[key]; ok {
		node.InUse += count
		c.moveToFront(node)
		return true, node.key
	}
	return false, "none"
}

// UpdateNode updates the usage count of a node.
func (c *LRUCache) UpdateNode(key string, count int) {
	if node, ok := c.cache[key]; ok {
		node.InUse += count
	}
}

// Put inserts a key into the current LRU cache.
// If it already exists, it moves it to the front.
// If it is full, it deletes the least recently used key.
func (c *LRUCache) Put(key string, count int) (bool, string) {
	deletedKey := "none"
	if node, ok := c.cache[key]; ok {
		c.moveToFront(node)
		return false, "alreadyExists"
	}

	newNode := &Node{key: key, InUse: count}
	if len(c.cache) >= c.capacity {
		// Cache is full and needs deletion.
		deletedKey = c.deleteNode(c.tail)
		if deletedKey == "none" {
			fmt.Println("All cache in use, go contact cloud")
			return false, "none"
		}
		// Can delete and add.
		delete(c.cache, deletedKey)
	}
	c.cache[key] = newNode
	c.addToFront(newNode)
	return true, deletedKey
}

func (c *LRUCache) moveToFront(node *Node) {
	if node == c.head {
		return
	}
	c.removeNode(node)
	c.addToFront(node)
}

// removeNode removes a node from the linked list.
func (c *LRUCache) removeNode(node *Node) {
	if node == nil {
		return
	}
	if node == c.head {
		c.head = node.next
		if c.head != nil {
			c.head.prev = nil
		}
	} else if node == c.tail {
		c.tail = node.prev
		if c.tail != nil {
			c.tail.next = nil
		}
	} else {
		if node.prev != nil {
			node.prev.next = node.next
		}
		if node.next != nil {
			node.next.prev = node.prev
		}
	}
	node.prev = nil
	node.next = nil
}

// deleteNode checks which node can be deleted and removes it.
func (c *LRUCache) deleteNode(node *Node) string {
	for node != nil && node.InUse >= 1 {
		node = node.prev
		if node == nil || node == c.head {
			return "none"
		}
	}
	if node == nil {
		return "none"
	}
	returnKey := node.key
	c.removeNode(node)
	return returnKey
}

// addToFront adds a node to the front of the linked list.
func (c *LRUCache) addToFront(node *Node) {
	if c.head == nil {
		c.head = node
		c.tail = node
	} else {
		node.next = c.head
		c.head.prev = node
		c.head = node
	}
}

// GetCacheList returns a list of keys in the cache.
func (c *LRUCache) GetCacheList() []string {
	current := c.head
	count := 0
	for current != nil {
		if count < len(c.buffer) {
			c.buffer[count] = current.key
		} else {
			c.buffer = append(c.buffer, current.key)
		}
		current = current.next
		count++
	}
	return c.buffer[:count]
}

// GetLength returns the current length of the cache.
func (c *LRUCache) GetLength() int {
	return len(c.cache)
}

// SizeOfSlice returns the size of a slice.
func SizeOfSlice(slice interface{}) uintptr {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		panic("SizeOfSlice: not a slice")
	}

	elemSize := v.Type().Elem().Size()
	return uintptr(v.Len()) * elemSize
}
