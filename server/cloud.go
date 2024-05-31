package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var cloudConcurrentConnection int = 0
var cloudConcurrentConnectionLock sync.Mutex
var cloudBandwidthPerConnection float64 = common.MaxBandwidth * 10
var cloudBandwidthLock sync.RWMutex

func SimulStartCloud() {
	router := gin.Default()
	router.POST("/getdata", simulCloudSendData)
	router.Run(fmt.Sprintf("%s:%s", common.ServerIP, common.CloudPort))
}

func simulCloudSendData(c *gin.Context) {
	currentSent := float64(0)
	cloudUpdateConcurrentConnection(1)
	for currentSent < common.DataSize {
		cloudBandwidthLock.RLock()
		currentSent += cloudBandwidthPerConnection
		cloudBandwidthLock.RUnlock()
		time.Sleep(1 * time.Second)
	}
	cloudUpdateConcurrentConnection(-1)
	c.Status(200)
}

func cloudUpdateConcurrentConnection(amount int) {
	defer cloudConcurrentConnectionLock.Unlock()
	cloudConcurrentConnectionLock.Lock()
	cloudConcurrentConnection += amount
	cloudUpdateBandwidthPerConnection()
}

func cloudUpdateBandwidthPerConnection() {
	defer cloudBandwidthLock.Unlock()
	cloudBandwidthLock.Lock()
	if cloudConcurrentConnection == 0 {
		cloudBandwidthPerConnection = common.MaxBandwidth * 10
	} else {
		cloudBandwidthPerConnection = common.MaxBandwidth / float64(cloudConcurrentConnection)
	}
}
