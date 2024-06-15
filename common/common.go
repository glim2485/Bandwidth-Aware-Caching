package common

import (
	"sync"
	"os/exec"
	"strings"
	"fmt"
)

var UserCount int = 50
var UserIterations int = 25
var EnableMulticast bool = true
var EnableCodeCache bool = true
var MaxBandwidth float64 = 2500 //MB aka 0.1GB
var DataSize float64 = 500      // MB aka 1GB
var ServerIP string = "localhost"
var MulticastIP string = "224.0.1.10"
var ServerPort string = "8080"
var CloudPort string = "60001"
var SwapItemSize int = 10
var UserCacheSize int = 25
var EdgeCacheSizeMultiplier int = 10
var EdgeCacheSize int = UserCacheSize * EdgeCacheSizeMultiplier
var SeedMultiplier int64 = 7
var MaxCodedItems int = 2 //must be equal or greater than 2
var MulticastCollectTime = 3
var MulticastBandwidthMultiplier = 0.5
var MaxFiles int = 50

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
	339: "Coded Multicast",
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

//used for debugging
func FindPortBind(port string) {
	cmd := exec.Command("lsof", "-i", "UDP:"+port)
			output, err := cmd.Output()
			if err != nil {
				fmt.Println("Error:", err)
			}
		
			// Output will be in the form of "<Command> <PID> ..."
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 2 && strings.HasSuffix(fields[8], ":"+port) {
					pid := fields[1]
					fmt.Printf("Process ID using port %s: %s\n",port, pid)
					// Optionally, you can get more details about the process using PID
					// e.g., parsing /proc/<pid>/cmdline for command line arguments
				}
			}
}