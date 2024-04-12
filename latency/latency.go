package latency

import (
	"gjlim2485/bandwidthawarecaching/common"
	"math"
	"time"
)

func CalculateExpectedTime(bandwidth int, datasize int) float64 {
	transferTime := float64(datasize) / float64(bandwidth)
	return transferTime

}

func DetermineTrigger(expectedTransferTime float64, realTransferTime float64) {
	if realTransferTime <= expectedTransferTime*common.MulticastTriggerModifier { //real transfer time is still within bounds, no need for trigger waiting
		common.TimerTime = time.Duration(0)
		common.ToggleCoding = false
	} else {
		waitTime := math.Abs(realTransferTime - expectedTransferTime)
		common.TimerTime = time.Duration(waitTime)
		common.ToggleCoding = true
	}
}
