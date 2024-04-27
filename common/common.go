package common

import (
	"sync"
	"time"
)

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
	Multicast bool
}

type UserIntersection struct {
	Users        []string `json: users`
	Intersection []string `json: intersection`
	RequestFile  []string `json: requestfile`
}

type CodedIntersection struct {
	Users        []string        `json: users`
	Intersection map[string]bool `json: intersection`
	CodedFile    map[string]bool `json: codedfiles`
}

var EdgeCache LRUCache

var UserNumbers int = 100
var UserRequestTicket int = 0
var UserRequestTicketResult = make(map[int][]UserIntersection)
var MulticastMutex sync.Mutex
var LatencyMutex sync.Mutex

var UserIteration int = 10
var UserLogInfo = make(map[int]UserLog)
var CloudIP string = "192.168.0.2"
var DataDirectory string = "/home/dnclab/Bandwidth-Aware-Caching/data"
var MyIP string = "192.168.0.2"
var ServerIP string = "192.168.0.2"
var ServerPort string = ":8080"
var MulticastIP string = "192.168.0.1"
var MulticastPort string = ":9999"
var GlobalTimer *time.Timer
var UserDataChannel = make(chan UserData, 30)
var ConnectedUsersTCP int = 0
var ConnectedUsersMulticast []ConnectedUsersUDP
var MinimumUserBandwidth float64 = 13.3
var ToggleMulticastMultiplier float64 = 1 //the higher, the higher the bar to trigger
var MulticastWaitTime int = 2000          //in milliseconds
var MaxLocalCacheSize int = 10
var MaxEdgeCacheSize int = 20
var CacheDataSize int = 5000 //in megabytes
var EnableCodeCache bool = true

func StringinSlice(str string, list []string) bool {
	for _, v := range list {
		if v == str {
			return true
		}
	}
	return false
}
