package latency

import "gjlim2485/bandwidthawarecaching/common"

func UpdateBandwidth() {
	totalConnection := common.ConnectedUsersTCP + len(common.ConnectedUsersMulticast)
	common.SplitBandwidth = common.TotalBandwidth / float64(totalConnection)
	if common.SplitBandwidth < common.ToggleMulticastMultiplier*common.MinimumUserBandwidth {
		common.ToggleMulticast = true
	} else {
		common.ToggleMulticast = false
	}
}

func UpdateWaitTime() {
	//TODO: how to calculate wait time
	common.MulticastWaitTime = 2000
}
