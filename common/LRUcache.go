package common

import (
	"fmt"
)

type LRUCache struct {
	capacity int
	cache    map[string]*Node
	head     *Node
	tail     *Node
}

type Node struct {
	key   string
	value string
	count int
	InUse bool
	prev  *Node
	next  *Node
}

var SimulFetchingData = make(map[string]bool)

func Constructor(capacity int) LRUCache {
	return LRUCache{
		capacity: capacity,
		cache:    make(map[string]*Node),
		head:     nil,
		tail:     nil,
	}
}

func (this *LRUCache) Get(key string) string {
	if node, ok := this.cache[key]; ok {
		this.moveToFront(node)
		return node.value
	}
	return ""
}

func (this *LRUCache) Put(key string, value string) {
	if node, ok := this.cache[key]; ok {
		node.value = value
		this.moveToFront(node)
	} else {
		newNode := &Node{key: key, value: value, InUse: true}
		if len(this.cache) >= this.capacity {
			delete(this.cache, this.tail.key)
			this.removeNode(this.tail)
		}
		this.cache[key] = newNode
		this.addToFront(newNode)
	}
}

func (this *LRUCache) PutEdge(key string, value string, size int) {
	didRemove := false
	if node, ok := this.cache[key]; ok {
		node.value = value
		this.moveToFront(node)
	} else {
		newNode := &Node{key: key, value: value, count: 1, InUse: true}
		if len(this.cache) >= this.capacity {
			if !this.cache[this.tail.key].InUse { //if the tail is not in use, remove it
				delete(this.cache, this.tail.key)
				this.removeNode(this.tail)
				didRemove = true
			} else {
				//if the tail is in use, remove the first non-in use node
				current := this.tail
				for current != nil {
					if !current.InUse {
						delete(this.cache, current.key)
						this.removeNode(current)
						didRemove = true
						break
					}
					current = current.prev
				}
			}
		}
		if didRemove {
			//simulate retrieving data from cloud
			SimulTransferringData(size)
			this.cache[key] = newNode
			this.addToFront(newNode)
		} else {
			fmt.Printf("All items cached currently in use, cannot add new item\n")
		}
	}
}

func (this *LRUCache) moveToFront(node *Node) {
	if node == this.head {
		return
	}
	this.removeNode(node)
	this.addToFront(node)
}

func (this *LRUCache) removeNode(node *Node) {
	if node == this.head {
		this.head = node.next
	} else if node == this.tail {
		this.tail = node.prev
	} else {
		node.prev.next = node.next
		node.next.prev = node.prev
	}
}

func (this *LRUCache) addToFront(node *Node) {
	if this.head == nil {
		this.head = node
		this.tail = node
	} else {
		node.next = this.head
		this.head.prev = node
		this.head = node
	}
}

func (this *LRUCache) GetCacheList() []string {
	current := this.head
	returnSlice := []string{}
	for current != nil {
		returnSlice = append(returnSlice, current.key)
		current = current.next
	}
	return returnSlice
}

func (this *LRUCache) ChangeFileStatus(status bool, filename string) {
	current := this.head
	for current.key != filename {
		current = current.next
		if current.key == filename {
			current.InUse = status
			break
		}
	}
}

func (this *LRUCache) SimulCheckFetchData(filename string, size int) bool {
	MulticastMutex.Lock()
	if SimulFetchingData[filename] {
		MulticastMutex.Unlock()
		return true
	}
	SimulUpdateConcurrentConnection(1)
	this.PutEdge(filename, filename, size)
	SimulUpdateConcurrentConnection(-1)
	SimulFetchingData[filename] = true
	MulticastMutex.Unlock()
	return true
}
