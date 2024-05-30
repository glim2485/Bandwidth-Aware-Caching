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

var transitDataCapacity int = 10
var transitDataList = make(map[string]int)
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
var swapItemLock sync.Mutex

type swapItemStruct struct {
	fullyCached bool
	inTransit   bool
	waitingSwap bool
}

// start server
func SimulStartServer(cacheSize int) {
	edgeCache = lru.Constructor(cacheSize)
	go dataCollector()
	router := gin.Default()
	router.Run(fmt.Sprintf("%s:%s", common.ServerIP, common.ServerPort))
	router.POST("/getdata", receiveRequest)
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
	defer concurrentConnectionLock.Unlock()
	concurrentConnectionLock.Lock()
	concurrentConnection += amount
	updateBandwidthPerConnection()
}

func updateBandwidthPerConnection() {
	defer bandwidthLock.Unlock()
	bandwidthLock.Lock()
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

func receiveRequest(c *gin.Context) {
	var userData common.UserRequest
	if err := c.BindJSON(&userData); err != nil {
		return
	}
	if !common.EnableMulticast {
		if multicastNeeded {
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
	if _, exists := swapItem[requestFile]; exists {

		canSwap := false
		//wait until a cache is stop being used to be swapped
		for !canSwap {
			time.Sleep(2 * time.Second)
			edgeCacheLock.Lock()
			swapItemLock.Lock()
			canSwap = edgeCache.Put(requestFile, true)
			if canSwap {
				delete(swapItem, requestFile)
				swapItem[requestFile] = true
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
		//does not exist in swap

		edgeCacheLock.Lock()
		edgeCache.Put(requestFile, true)
		edgeCacheLock.Unlock()
	}
}
