package codecache

type LRUCache struct {
	capacity int
	cache    map[string]*Node
	head     *Node
	tail     *Node
}

type Node struct {
	key   string
	value string
	prev  *Node
	next  *Node
}

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
		newNode := &Node{key: key, value: value}
		if len(this.cache) >= this.capacity {
			delete(this.cache, this.tail.key)
			this.removeNode(this.tail)
		}
		this.cache[key] = newNode
		this.addToFront(newNode)
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
