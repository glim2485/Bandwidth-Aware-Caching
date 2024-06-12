package common

import "sync"

var UserCount int = 50
var UserIterations int = 25
var EnableMulticast bool = true
var EnableCodeCache bool = false
var MaxBandwidth float64 = 2500 //MB aka 0.1GB
var DataSize float64 = 100      // MB aka 1GB
var ServerIP string = "localhost"
var MulticastIP string = "224.0.1.10"
var ServerPort string = "8080"
var CloudPort string = "55555"
var SwapItemSize int = 10
var UserCacheSize int = 50
var EdgeCacheSize int = UserCacheSize * 10

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
}
