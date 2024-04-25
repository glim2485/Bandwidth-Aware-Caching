package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/codecache"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/latency"
	"gjlim2485/bandwidthawarecaching/lrucache"
	"net"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

var SimulChan = make(chan []common.UserIntersection)
var collectChan = make(chan common.UserData, 20)

func SimulInitializeServer() {
	common.EdgeCache = lrucache.Constructor(common.MaxEdgeCacheSize)
	go SimulMulticastDataCollector()
}

func SimulIncomingData(userID int, filename string, userCache []string) (bool, int, float64) {
	if latency.ToggleMulticast {
		collectChan <- common.UserData{UserIP: strconv.Itoa(userID), LocalCache: userCache, RequestData: filename}
		myTicket := common.UserRequestTicket
		//perform blocking wait
		for common.UserRequestTicketResult[myTicket] == nil {
			//wait for the data to come back
		}
		var result = common.UserRequestTicketResult[myTicket]
		if !common.EnableCodeCache {
			//no code cache, so no mulple requests
			for _, s := range result {
				if s.RequestFile[0] == filename {
					if common.EdgeCache.Get(filename) != "" {
						//cache was hit
						return true, common.CacheDataSize, (float64(1) / float64(len(s.Users)))
					} else {
						//cache miss
						common.EdgeCache.SimulCheckFetchData(filename, common.CacheDataSize) //this blocks all processes until the data is fetched
						return false, common.CacheDataSize, (float64(1) / float64(len(s.Users)))
					}
				}
			}
		} else {
			//code cache enabled
			for _, s := range result {
				if common.StringinSlice(filename, s.RequestFile) {
					if s.Intersection[0] == "miss" {
						common.EdgeCache.SimulCheckFetchData(filename, common.CacheDataSize)
						return false, common.CacheDataSize, (float64(1) / float64(len(s.Users)))
					} else {
						common.EdgeCache.Get(filename) //this is done to refresh cache
						return true, common.CacheDataSize, (float64(1) / float64(len(s.Users)))
					}
				}
			}
		}
	} else {
		//unicast scenario
		hit := common.EdgeCache.Get(filename)
		if hit != "" {
			//cache was hit
			latency.SimulUpdateConcurrentConnection(1)
			return true, common.CacheDataSize, 1
		} else {
			//cache miss
			latency.SimulUpdateConcurrentConnection(1) //simulate edge to cache fetch
			common.EdgeCache.PutEdge(filename, filename, common.CacheDataSize)
			latency.SimulUpdateConcurrentConnection(-1) //undo connection
			return false, common.CacheDataSize, 1
		}
	}
	//error
	return false, 0, 0
}

func SimulMulticastDataCollector() {
	timer := time.NewTicker(time.Duration(common.MulticastWaitTime) * time.Millisecond)
	var collectedData []common.UserData
	for {
		select {
		case <-timer.C:
			//keep emptying the channel
			for len(collectChan) > 0 {
				incomingData := <-collectChan
				collectedData = append(collectedData, incomingData)
			}
		default:
			//do something with said data
			copiedData := make([]common.UserData, len(collectedData))
			copy(copiedData, collectedData)
			go codecache.MakeGroups(copiedData)
			common.UserRequestTicket++
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
