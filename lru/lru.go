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

// creates an LRU cache with the given capacity
func Constructor(capacity int) LRUCache {
	return LRUCache{
		capacity: capacity,
		cache:    make(map[string]*Node),
		head:     nil,
		tail:     nil,
		buffer:   make([]string, 0, capacity),
	}
}

// gets a key from the current LRUcache
// returns true and the value of the key if it exists
// returns false and "none" if it does not
func (c *LRUCache) Get(key string, count int) (bool, string) {
	if node, ok := c.cache[key]; ok {
		node.InUse = node.InUse + count
		c.moveToFront(node)
		return true, node.key
	}
	return false, "none"
}

// puts a key into the current LRUcache
// if it already exists, it moves it to the front
// if it is full, it deletes the least recently used key
// USED BY USERS

func (c *LRUCache) UpdateNode(key string, count int) {
	if node, ok := c.cache[key]; ok {
		node.InUse = node.InUse + count
	}
}
func (c *LRUCache) Put(key string, count int) (bool, string) {
	deletedKey := "none"
	if node, ok := c.cache[key]; ok {
		c.moveToFront(node)
		return false, "alreadyExists"
	} else {
		newNode := &Node{key: key, InUse: count}
		if len(c.cache) >= c.capacity {
			//cache full needs deletion
			deletedKey = c.deleteNode(c.tail)
			if deletedKey == "none" {
				//can't delete anything
				fmt.Println("all cache in use, go contact cloud")
				return false, "none"
			}
			//can delete and add
			delete(c.cache, deletedKey)
		}
		c.cache[key] = newNode
		c.addToFront(newNode)
		return true, deletedKey
	}
}

func (c *LRUCache) moveToFront(node *Node) {
	if node == c.head {
		return
	}
	c.removeNode(node)
	c.addToFront(node)
}

// used for moving nodes around
func (c *LRUCache) removeNode(node *Node) {
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
		node.prev.next = node.next
		node.next.prev = node.prev
	}
	node.prev = nil
	node.next = nil
}

// checks which node can be deleted
func (c *LRUCache) deleteNode(node *Node) string {
	for node.InUse >= 1 {
		node = node.prev
		if node == nil || c.head == node {
			return "none"
		}
	}
	returnKey := node.key
	c.removeNode(node)
	return returnKey
}

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

func (c *LRUCache) GetCacheList() []string {
	current := c.head
	count := 0
	for current != nil {
		if count < len(c.buffer) {
			c.buffer[count] = current.key
		} else {
			c.buffer = append(c.buffer, current.key)
			//fmt.Println("buffer size", SizeOfSlice(c.buffer), " and current key size", unsafe.Sizeof(current.key))
		}
		current = current.next
		count++
	}
	return c.buffer[:count]
}

//debug use

func (c *LRUCache) GetLength() int {
	return len(c.cache)
}

func SizeOfSlice(slice interface{}) uintptr {
	v := reflect.ValueOf(slice)
	if v.Kind() != reflect.Slice {
		panic("SizeOfSlice: not a slice")
	}

	elemSize := v.Type().Elem().Size()
	return uintptr(v.Len()) * elemSize
}
