package common

import "time"

type UserData struct {
	UserIP      string   `json:userip`
	LocalCache  []string `json:localcache`
	RequestData string   `json:requestdata`
}

type ConnectedUsersUDP struct {
	UserIP      []string
	MulticastIP string
}

type UserLog struct {
	UserResult []UserCacheHit
}

type UserCacheHit struct {
	ItemName  string
	CacheHit  string
	TimeTaken int
}

var UserNumbers int = 100
var UserIteration int = 100
var UserLogInfo = make(map[int]UserLog)
var CloudIP string = "192.168.0.2"
var DataDirectory string = "/home/dnclab/Bandwidth-Aware-Caching/data"
var MyIP string = "192.168.0.2"
var ServerIP string = "192.168.0.2"
var ServerPort string = ":8080"
var MulticastIP string = "192.168.0.1"
var MulticastPort string = ":9999"
var MulticastTriggerModifier float64 = 0.8 //lower makes the bounds tighter
var ToggleMulticast bool = false
var TimerTime time.Duration
var GlobalTimer *time.Timer
var UserDataChannel = make(chan UserData, 30)
var TotalBandwidth float64 = 1000
var ConnectedUsersTCP int = 0
var ConnectedUsersMulticast []ConnectedUsersUDP
var SplitBandwidth float64
var MinimumUserBandwidth float64 = 13.3
var ToggleMulticastMultiplier float64 = 1 //the higher, the higher the bar to trigger
var MulticastWaitTime int = 2000          //in milliseconds
var MaxLocalCacheSize int = 10
var MaxEdgeCacheSize int = 20
var CacheDataSize int = 5000 //in megabytes
