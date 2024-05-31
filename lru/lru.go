package lru

import "fmt"

type LRUCache struct {
	capacity int
	cache    map[string]*Node
	head     *Node
	tail     *Node
}

type Node struct {
	key   string
	InUse bool
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
	}
}

// gets a key from the current LRUcache
// returns true and the value of the key if it exists
// returns false and "none" if it does not
func (c *LRUCache) Get(key string, status bool) (bool, string) {
	if node, ok := c.cache[key]; ok {
		node.InUse = status
		c.moveToFront(node)
		return true, node.key
	}
	return false, "none"
}

// puts a key into the current LRUcache
// if it already exists, it moves it to the front
// if it is full, it deletes the least recently used key
// USED BY USERS

func (c *LRUCache) UpdateNode(key string, status bool) {
	if node, ok := c.cache[key]; ok {
		node.InUse = status
	}
}
func (c *LRUCache) Put(key string, status bool) (bool, string) {
	deletedKey := "none"
	if node, ok := c.cache[key]; ok {
		c.moveToFront(node)
		return true, "none"
	} else {
		newNode := &Node{key: key, InUse: status}
		if len(c.cache) >= c.capacity {
			deletedKey = c.deleteNode(c.tail)
			if deletedKey == "none" {
				fmt.Println("all cache in use, go contact cloud")
				return false, "none"
			}
			delete(c.cache, deletedKey)
		}
		c.cache[key] = newNode
		c.addToFront(newNode)
	}
	return true, deletedKey
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
	} else if node == c.tail {
		c.tail = node.prev
	} else {
		node.prev.next = node.next
		node.next.prev = node.prev
	}
}

func (c *LRUCache) deleteNode(node *Node) string {
	for node.InUse {
		node = node.prev
		if c.head == node {
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
	returnSlice := []string{}
	for current != nil {
		returnSlice = append(returnSlice, current.key)
		current = current.next
	}
	return returnSlice
}
