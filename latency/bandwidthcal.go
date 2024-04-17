package latency

import "gjlim2485/bandwidthawarecaching/common"

func UpdateBandwidth() {
	totalConnection := common.ConnectedUsersTCP + len(common.ConnectedUsersMulticast)
	common.SplitBandwidth = common.TotalBandwidth / float64(totalConnection)
	//TODO, check if it matches minimum bandwidth
}
