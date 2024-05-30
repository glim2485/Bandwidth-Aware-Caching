package common

var EnableMulticast bool = false
var EnableCodeCache bool = false
var MaxBandwidth float64 = 100 //MB aka 0.1GB
var ServerIP string = "localhost"
var MulticastIP string = "224.0.1.10"
var ServerPort string = "8080"
var DataSize float64 = 1000 // MB aka 1GB

// shared structs for users to encode and decode JSON files
type UserRequest struct {
	UserID      int      `json:"UserID"`
	RequestFile string   `json:"RequestFile"`
	UserFile    []string `json:"UserFile"`
}
