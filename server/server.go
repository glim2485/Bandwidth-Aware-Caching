package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/lru"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var concurrentConnection int = 0
var concurrentConnectionLock sync.Mutex
var bandwidthPerConnection float64 = common.MaxBandwidth
var bandwidthLock sync.RWMutex
var multicastNeeded bool = false
var udpAnnounceChannel = make(map[int]chan string)
var announceChannelLock sync.Mutex
var incomingData = make(chan common.UserRequest, 30)
var edgeCache lru.LRUCache
var edgeCacheLock sync.Mutex
var swapItem = make(map[string]swapItemStruct)
var swapItemCapacity int = common.SwapItemSize
var swapItemLock sync.Mutex

type swapItemStruct struct {
	fullyCached bool //the entire file exists in the server now
	inTransit   bool //file was called and is being sent to the edge for caching then to user
	waitingSwap bool //all cache is full and is waiting to be swapped
}

// start server
func SimulStartServer() {
	edgeCache = lru.Constructor(common.EdgeCacheSize)
	go dataCollector()
	router := gin.Default()

	router.POST("/getdata", receiveRequest)
	router.Run(fmt.Sprintf("%s:%s", common.ServerIP, common.ServerPort))
}

// this should be done as a SINGULAR go routine
func dataCollector() {
	ticker := time.NewTicker(multicastWaitTime)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			//use collectedData
			copiedData := make([]common.UserRequest, len(collectedData))
			copy(copiedData, collectedData)
			//make sure to pass by value
			go handleData(copiedData, currUDPPort)
			//reset collectedData
			collectedData = make([]common.UserRequest, 0)
		case data := <-incomingData:
			collectedData = append(collectedData, data)
		}
	}
}

func updateConcurrentConnection(amount int) {
	concurrentConnectionLock.Lock()
	defer concurrentConnectionLock.Unlock()
	concurrentConnection += amount
	updateBandwidthPerConnection()
}

func updateBandwidthPerConnection() {
	bandwidthLock.Lock()
	defer bandwidthLock.Unlock()
	if concurrentConnection == 0 {
		bandwidthPerConnection = common.MaxBandwidth
	} else {
		bandwidthPerConnection = common.MaxBandwidth / float64(concurrentConnection)
		//update multicast needed
		if bandwidthPerConnection < 0.2*common.MaxBandwidth {
			multicastNeeded = true
		} else {
			multicastNeeded = false
		}
	}
}

// all apicalls
func simulSendData(numconn int) {
	currentSent := float64(0)
	updateConcurrentConnection(numconn)
	for currentSent < common.DataSize {
		bandwidthLock.RLock()
		currentSent += bandwidthPerConnection
		bandwidthLock.RUnlock()
		time.Sleep(1 * time.Second)
	}
	updateConcurrentConnection(-numconn)
}

// to be used for cloud -> edge
func simulFetchData(requestFile string, c *gin.Context) {
	currentReceived := float64(0)
	updateConcurrentConnection(1)
	for currentReceived < common.DataSize {
		bandwidthLock.RLock()
		currentReceived += bandwidthPerConnection
		bandwidthLock.RUnlock()
		time.Sleep(1 * time.Second)
	}
	updateConcurrentConnection(-1)
	canSwap := false
	//wait until a cache is stop being used to be swapped
	for !canSwap {
		time.Sleep(2 * time.Second)
		edgeCacheLock.Lock()
		swapItemLock.Lock()
		canSwap, removedItem := edgeCache.Put(requestFile, true)
		if canSwap {
			delete(swapItem, requestFile)
			newItem := swapItemStruct{fullyCached: true, inTransit: true, waitingSwap: true}
			swapItem[removedItem] = newItem
		}
		swapItemLock.Unlock()
		edgeCacheLock.Unlock()
	}
	simulSendData(1)
	edgeCacheLock.Lock()
	edgeCache.UpdateNode(requestFile, false)
	edgeCacheLock.Unlock()
	c.Status(336)
}

func receiveRequest(c *gin.Context) {
	var userData common.UserRequest
	if err := c.BindJSON(&userData); err != nil {
		return
	}
	if !common.EnableMulticast {
		if multicastNeeded {
			//multicast data
			incomingData <- userData
			userPort := fetchUDPPort(userData.UserID)
			response := gin.H{
				"UDPPort": userPort,
				"status":  "change to Multicast",
			}
			c.JSON(333, response)
		} else {
			sendUnicastData(userData.RequestFile, c)
		}
	} else {
		simulSendData(1)
		c.Status(http.StatusOK)
	}
}

func fetchUDPPort(userid int) string {
	udpAnnounceChannel[userid] = make(chan string)
	returnPort := <-udpAnnounceChannel[userid]
	announceChannelLock.Lock()
	defer announceChannelLock.Unlock()
	close(udpAnnounceChannel[userid])
	delete(udpAnnounceChannel, userid)
	return returnPort
}

func sendUnicastData(requestFile string, c *gin.Context) {
	edgeCacheLock.Lock()
	exists, _ := edgeCache.Get(requestFile, true)
	if exists {
		//exists in cache, http 200 for hit
		edgeCacheLock.Unlock()
		simulSendData(1)
		edgeCacheLock.Lock()
		edgeCache.UpdateNode(requestFile, false)
		edgeCacheLock.Unlock()
		c.Status(http.StatusOK)
	} else {
		//does not exist cache, check swap
		edgeCacheLock.Unlock()
		swapCacheAndSwap(requestFile, c)
	}
}

func swapCacheAndSwap(requestFile string, c *gin.Context) {
	swapItemLock.Lock()
	if _, exists := swapItem[requestFile]; exists {
		swap := swapItem[requestFile]
		swap.waitingSwap = true
		swapItem[requestFile] = swap
		swapItemLock.Unlock()
		canSwap := false
		//wait until a cache is stop being used to be swapped
		for !canSwap {
			time.Sleep(2 * time.Second)
			edgeCacheLock.Lock()
			swapItemLock.Lock()
			canSwap, removedItem := edgeCache.Put(requestFile, true)
			if canSwap {
				delete(swapItem, requestFile)
				newItem := swapItemStruct{fullyCached: true, inTransit: true, waitingSwap: true}
				swapItem[removedItem] = newItem
			}
			swapItemLock.Unlock()
			edgeCacheLock.Unlock()
		}
		simulSendData(1)
		edgeCacheLock.Lock()
		edgeCache.UpdateNode(requestFile, false)
		edgeCacheLock.Unlock()
		//exists in swap, http 335 for swap
		c.Status(335)
	} else {
		//does not exist in swap nor in cache
		if len(swapItem) >= swapItemCapacity {
			newItem := swapItemStruct{fullyCached: false, inTransit: true, waitingSwap: true}
			swapItem[requestFile] = newItem
			go simulFetchData(requestFile, c)
			swapItemLock.Unlock()
			return
		}
		removeItem := checkSwapItem()
		//if there is a replacement item, remove it and add the new item
		if removeItem != "none" {
			delete(swapItem, removeItem)
			newItem := swapItemStruct{fullyCached: false, inTransit: true, waitingSwap: true}
			swapItem[requestFile] = newItem
			//simulate fetching data from cloud and swapping
			go simulFetchData(requestFile, c)
		} else {
			//no target at all, client needs to do a direct fetch to cloud
			c.Status(334)
		}
		swapItemLock.Unlock()
	}
}

func checkSwapItem() string {
	for x := range swapItem {
		//if there exists a swap item that is not in transit and not waiting to be swapped
		//replace that one
		if !swapItem[x].waitingSwap && !swapItem[x].inTransit {
			return x
		}
	}
	// all swap items are in transit or waiting to be swapped
	return "none"
}
