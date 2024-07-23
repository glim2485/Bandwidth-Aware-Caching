package datamovement

var CurrentConnection int
var MaxBandwidth float64
var BandwidthPerUser float64

func SimulUpdateUserBandwidth() {
	BandwidthPerUser = MaxBandwidth / float64(CurrentConnection)
}
