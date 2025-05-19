package bandmonitor

import (
	"gjlim2485/bandwidthawarecaching/common"
	"sync"
	"time"
)

var currentConnection float64 = 0.0
var maxBandwidth float64
var BandwidthPerUser float64 = 0.0
var bandwidthLock sync.RWMutex
var concurrentLock sync.RWMutex

func BandInit() {
	maxBandwidth = common.MaxBandwidth
	go bandwidthMonitor()
}

func bandwidthMonitor() {
	for {
		concurrentLock.RLock()
		bandwidthLock.Lock()
		if currentConnection >= 1 {
			BandwidthPerUser = maxBandwidth / currentConnection
		} else {
			BandwidthPerUser = maxBandwidth
		}
		concurrentLock.RUnlock()
		bandwidthLock.Unlock()
		time.Sleep(500 * time.Millisecond)
	}
}

func UpdateUserCount(count float64) {
	concurrentLock.Lock()
	defer concurrentLock.Unlock()
	currentConnection += count
}

func GetCurrentBandwidth() float64 {
	bandwidthLock.RLock()
	defer bandwidthLock.RUnlock()
	return BandwidthPerUser
}

func LogBandwidth(exitChannel chan int, returnChannel chan float64, sleepTimer int) {
	time.Sleep(time.Duration(sleepTimer) * time.Millisecond)
	count := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	updatingAverage := float64(0)
	for {
		select {
		case <-ticker.C:
			count++
			currentBandwidth := GetCurrentBandwidth()
			if currentBandwidth >= 0 {
				updatingAverage = updatingAverage + (currentBandwidth-updatingAverage)/float64(count)
			} else {
				count-- //ignore this count
			}
		case <-exitChannel:
			returnChannel <- updatingAverage
			return
		}
	}
}
