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
	for i := 0; i < common.EdgeCacheSize; i++ {
		edgeCache.Put(fmt.Sprintf("file%d", (i+1)), 1)
	}
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
			collectedData = collectedData[:0]
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
		fmt.Println("Server: Sending data to some client progress ", currentSent, "/", common.DataSize)
		time.Sleep(1 * time.Second)
	}
	updateConcurrentConnection(-numconn)
}

// to be used for cloud -> edge
func simulFetchData(requestFile string, c *gin.Context, returnCode int) {
	fmt.Println("Server: Fetching data from cloud for", requestFile)
	currentReceived := float64(0)
	updateConcurrentConnection(1)
	for currentReceived < common.DataSize {
		bandwidthLock.RLock()
		currentReceived += bandwidthPerConnection
		bandwidthLock.RUnlock()
		time.Sleep(1 * time.Second)
		fmt.Println("Server:", requestFile, "progress:", currentReceived, "/", common.DataSize)
	}
	updateConcurrentConnection(-1)
	fmt.Println("Server: Data fetched from cloud for", requestFile)
	canSwap := false
	//wait until a cache is stop being used to be swapped
	fmt.Println("Server: checking for suitable cache replacement")
	var removedItem string
	for !canSwap {
		//TODO: for some reason, this loop is not exiting
		edgeCacheLock.Lock()
		swapItemLock.Lock()
		canSwap, removedItem = edgeCache.Put(requestFile, 1)
		if canSwap {
			if removedItem == "none" {
				fmt.Println("Server: added to cache", requestFile, "Can swap:", canSwap)
			} else {
				delete(swapItem, requestFile)
				newItem := swapItemStruct{fullyCached: true, inTransit: false, waitingSwap: false}
				swapItem[removedItem] = newItem
				fmt.Println("Server: cache replacement", removedItem, "for", requestFile, "Can swap:", canSwap)
			}
		}
		swapItemLock.Unlock()
		edgeCacheLock.Unlock()
	}
	fmt.Println("Server: Sending", requestFile, "to client")
	simulSendData(1)
	edgeCacheLock.Lock()
	edgeCache.UpdateNode(requestFile, -1)
	edgeCacheLock.Unlock()
	c.Status(returnCode)
}

func receiveRequest(c *gin.Context) {
	var userData common.UserRequest
	if err := c.BindJSON(&userData); err != nil {
		return
	}
	fmt.Println("Server: Received request from user", userData.UserID, "for data", userData.RequestFile)
	if common.EnableMulticast {
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
			fmt.Println("Server: Multicast not needed, fetching data directly for", userData.UserID)
			sendUnicastData(userData.RequestFile, c)
		}
	} else {
		fmt.Println("Server: Multicast not needed, fetching data directly for", userData.UserID)
		sendUnicastData(userData.RequestFile, c)
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
	exists, _ := edgeCache.Get(requestFile, 1)
	if exists {
		//exists in cache, http 200 for hit
		fmt.Println("Server cache hit for", requestFile)
		edgeCacheLock.Unlock()
		simulSendData(1)
		edgeCacheLock.Lock()
		edgeCache.UpdateNode(requestFile, -1)
		edgeCacheLock.Unlock()
		c.Status(http.StatusOK)
	} else {
		//does not exist cache, check swap
		fmt.Println("Server cache miss for", requestFile)
		edgeCacheLock.Unlock()
		swapCacheAndSwap(requestFile, c)
	}
}

func swapCacheAndSwap(requestFile string, c *gin.Context) {
	swapItemLock.Lock()
	if _, exists := swapItem[requestFile]; exists {
		//if exists in swap
		fmt.Println("Server: swap hit for", requestFile)
		if swapItem[requestFile].inTransit && swapItem[requestFile].waitingSwap {
			fmt.Println("Server: swap hit for", requestFile, "is already in transit due to another process")
			//someone else already called for this, just need to wait
			swapItemLock.Unlock()
			inEdge := false
			for !inEdge {
				edgeCacheLock.Lock()
				inEdge, _ = edgeCache.Get(requestFile, 1)
				edgeCacheLock.Unlock()
			}
			simulSendData(1)
			edgeCacheLock.Lock()
			edgeCache.UpdateNode(requestFile, -1)
			edgeCacheLock.Unlock()
			c.Status(338)
			return
		}
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
			canSwap, removedItem := edgeCache.Put(requestFile, 1)
			if canSwap {
				delete(swapItem, requestFile)
				newItem := swapItemStruct{fullyCached: true, inTransit: false, waitingSwap: false}
				swapItem[removedItem] = newItem
				fmt.Println("Server swap ", removedItem, "for", requestFile)
			}
			swapItemLock.Unlock()
			edgeCacheLock.Unlock()
		}
		simulSendData(1)
		edgeCacheLock.Lock()
		edgeCache.UpdateNode(requestFile, -1)
		edgeCacheLock.Unlock()
		//exists in swap, http 335 for swap
		c.Status(335)
	} else {
		//does not exist in swap
		//if there is enough space in swap, just call
		fmt.Println("Server swap miss for", requestFile)
		if len(swapItem) <= swapItemCapacity {
			newItem := swapItemStruct{fullyCached: false, inTransit: true, waitingSwap: true}
			swapItem[requestFile] = newItem
			fmt.Println("Server swap memory NOT full, requesting", requestFile)
			swapItemLock.Unlock()
			simulFetchData(requestFile, c, 336)
			return
		} else {
			//if there is no empty space in swap
			fmt.Println("Server swap memory FULL")
			removeItem := checkSwapItem()
			if removeItem != "none" {
				//there is a removable swap item
				delete(swapItem, removeItem)
				newItem := swapItemStruct{fullyCached: false, inTransit: true, waitingSwap: true}
				swapItem[requestFile] = newItem
				fmt.Println("Server swapping swap memory", removeItem, " for", requestFile)
				swapItemLock.Unlock()
				simulFetchData(requestFile, c, 337)
			} else {
				//no swap is replacable, need to fetch from cloud
				fmt.Println("Server no swap memory to replace, fetching from cloud")
				swapItemLock.Unlock()
				c.Status(334)
				return
			}
		}
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
