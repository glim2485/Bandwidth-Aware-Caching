package latency

import (
	"math"
	"sync"
	"time"
)

var mutex sync.Mutex
var SplitBandwidth float64
var TotalBandwidth float64 = 1000
var SimulUserConnected float64 = 0
var MulticastTriggerModifier float64 = 0.8 //lower makes the bounds tighter
var ToggleMulticast bool = false
var TimerTime time.Duration

func SimulTransferringData(filesize int) int {
	transferred_data := 0
	currentTime := time.Now()
	for transferred_data < filesize {
		transferred_data += int(SplitBandwidth)
		time.Sleep(1 * time.Second)
	}
	newTime := time.Now()
	timeTaken := newTime.Sub(currentTime)
	timeTakenInt := int(timeTaken.Milliseconds())
	return timeTakenInt
}

func SimulUpdateConcurrentConnection(amount float64) {
	mutex.Lock()
	SimulUserConnected += amount
	if SimulUserConnected >= 1 {
		SplitBandwidth = TotalBandwidth / SimulUserConnected
	} else {
		SplitBandwidth = TotalBandwidth
	}
	mutex.Unlock()
}

func CalculateExpectedTime(bandwidth int, datasize int) float64 {
	transferTime := float64(datasize) / float64(bandwidth)
	return transferTime

}

func DetermineTrigger(expectedTransferTime float64, realTransferTime float64) {
	if realTransferTime <= expectedTransferTime*MulticastTriggerModifier { //real transfer time is still within bounds, no need for trigger waiting
		TimerTime = time.Duration(0)
		ToggleMulticast = false
	} else {
		waitTime := math.Abs(realTransferTime - expectedTransferTime)
		TimerTime = time.Duration(waitTime)
		ToggleMulticast = true
	}
}
