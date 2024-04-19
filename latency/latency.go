package latency

import (
	"gjlim2485/bandwidthawarecaching/common"
	"math"
	"time"
)

func SimulTransferringData(filesize int) int {
	transferred_data := 0
	currentTime := time.Now()
	for transferred_data < filesize {
		transferred_data += int(common.SplitBandwidth)
		time.Sleep(1 * time.Second)
	}
	newTime := time.Now()
	timeTaken := newTime.Sub(currentTime)
	timeTakenInt := int(timeTaken.Milliseconds())
	return timeTakenInt
}

func CalculateExpectedTime(bandwidth int, datasize int) float64 {
	transferTime := float64(datasize) / float64(bandwidth)
	return transferTime

}

func DetermineTrigger(expectedTransferTime float64, realTransferTime float64) {
	if realTransferTime <= expectedTransferTime*common.MulticastTriggerModifier { //real transfer time is still within bounds, no need for trigger waiting
		common.TimerTime = time.Duration(0)
		common.ToggleMulticast = false
	} else {
		waitTime := math.Abs(realTransferTime - expectedTransferTime)
		common.TimerTime = time.Duration(waitTime)
		common.ToggleMulticast = true
	}
}
