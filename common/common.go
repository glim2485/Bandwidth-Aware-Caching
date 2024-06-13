package common

import "sync"

var UserCount int = 50
var UserIterations int = 25
var EnableMulticast bool = false
var EnableCodeCache bool = false
var MaxBandwidth float64 = 2500 //MB aka 0.1GB
var DataSize float64 = 500      // MB aka 1GB
var ServerIP string = "localhost"
var MulticastIP string = "224.0.1.10"
var ServerPort string = "8080"
var CloudPort string = "55555"
var SwapItemSize int = 10
var UserCacheSize int = 25
var EdgeCacheSize int = UserCacheSize * 10
var SeedMultiplier int64 = 7
var MaxCodedItems int = 2 //must be equal or greater than 2
var MulticastCollectTime = 3
var MulticastBandwidthMultiplier = 0.5

// shared structs for users to encode and decode JSON files
type UserRequest struct {
	UserID      int      `json:"UserID"`
	RequestFile string   `json:"RequestFile"`
	UserFile    []string `json:"UserFile"`
}

// for logging purposes
var UserDataLog []UserDataLogStruct
var UserDataLogLock sync.Mutex

type UserDataLogStruct struct {
	UserID      int    `json:"UserID"`
	RequestFile string `json:"RequestFile"`
	ReturnCode  int    `json:"ReturnCode"`
	FetchType   string `json:"FetchType"`
	TimeTaken   int    `json:"TimeTaken"`
}

var FetchType = map[int]string{
	200: "Unicast",
	333: "Multicast",
	334: "Cloud",
	335: "Swap",
	336: "In-Transit",
	337: "Cloud and swap",
	338: "Another one was fetching first",
	000: "device cache hit",
}

func SliceContainsInt(slice []int, item int) bool {
	for _, x := range slice {
		if x == item {
			return true
		}
	}
	return false
}

func SliceContainsString(slice []string, item string) bool {
	for _, x := range slice {
		if x == item {
			return true
		}
	}
	return false
}
