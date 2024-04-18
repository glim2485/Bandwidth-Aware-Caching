package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/codecache"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/latency"
	"net"
	"time"

	"github.com/gin-gonic/gin"
)

var EdgeCache codecache.LRUCache

var resultChan = make(chan []common.UserIntersection)
var collectChan = make(chan common.UserData, 20)

func SimulInitializeServer() {
	EdgeCache = codecache.Constructor(common.MaxEdgeCacheSize)
	go MulticastDataCollector()
}

func SimulIncomingData(userID int, filename string, userCache []string) (bool, int) {
	checkEdgeCache := EdgeCache.Get(filename)
	if checkEdgeCache != "" { //hit
		if common.ToggleMulticast { //if multicast is on, collect data instead of returning
			collectChan <- common.UserData{userIP: userID, LocalCache: userCache,RequestData: filename}

		} else {
			//TODO: dummy values for now
			latency.UpdateBandwidth()
			return true, common.CacheDataSize
		}
	} else {
		return false, 0
	}
}

func MulticastDataCollector() {
	timer := time.NewTicker(time.Duration(common.MulticastWaitTime) * time.Millisecond)
	var collectedData []common.UserData
	for {
		select {
		case <-resultChan:
			//got previous request back

		case <-timer.C:
			//keep collecting data
			for len(collectChan) > 0 {
				incomingData := <-collectChan
				collectedData = append(collectedData, incomingData)
			}
		default:
			//do something with said data
			go codecache.MakeGroups(collectedData, resultChan)
			collectedData = nil
		}
	}

}

func HTTPServer() {
	router := gin.Default()

	if err := router.Run(common.ServerPort); err != nil {
		fmt.Println("Error starting HTTP server")
		return
	}

	router.Run("0.0.0.0:8080")

}

func MulticastServer(portNumber int) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(common.MulticastIP),
		Port: portNumber,
	})
	if err != nil {
		fmt.Println("Error creating UDP connection:", err)
		return
	}
	defer conn.Close()
}

func CollectData() {
	var dataCollection []common.UserData
	for {
		select {
		case <-common.GlobalTimer.C:
			if len(dataCollection) != 0 {
				//check for relations
			}
			dataCollection = nil //reset the
		default:
			incomingData := <-common.UserDataChannel
			dataCollection = append(dataCollection, incomingData)
		}
	}
}
