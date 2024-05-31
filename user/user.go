package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/lru"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strconv"
	"sync"
)

type serverResponse struct {
	Port          string `json:"Port"`
	StatusMessage string `json:"StatusMessage"`
}

func SimulUserRequests(userid int, iteration int, cacheSize int, wg *sync.WaitGroup) {
	defer wg.Done()
	//set its port to be 40000 + userid
	ownPort := strconv.Itoa(40000 + userid)
	//all user cache inuse is set to false for experiment's sake
	userCache := lru.Constructor(cacheSize)
	for i := 0; i < iteration; i++ {
		userFiles := userCache.GetCacheList()
		userRequest := generateRequestFile()
		requestMessage := common.UserRequest{
			UserID:      userid,
			RequestFile: userRequest,
			UserFile:    userFiles,
		}
		jsonData, err := json.Marshal(requestMessage)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return
		}

		url := "http://" + common.ServerIP + ":" + common.ServerPort + "/getdata"
		body := bytes.NewBuffer(jsonData)
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return
		}
		defer resp.Body.Close()

		switch resp.StatusCode {
		case 200:
			fmt.Println("Request successful")
			userCache.Put(userRequest, false)
		case 333:
			fmt.Println("change to multicast")
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				fmt.Println("Error reading response body:", err)
				return
			}
			var response serverResponse
			err = json.Unmarshal(body, &response)
			if err != nil {
				fmt.Println("Error unmarshalling JSON:", err)
				return
			}
			if joinMulticast(response.Port, ownPort) {
				userCache.Put(userRequest, false)
			} else {
				fmt.Println("Error joining multicast group")
			}
		case 334:
			fmt.Println("direct fetch from cloud")
		}
	}
}

// case 335: was swapped with swapped item
// case 336: cache needs to be fetched from cloud, in-transit
func generateRequestFile() string {
	return "file" + strconv.Itoa(rand.Intn(50)+1)
}

func joinMulticast(port string, ownPort string) bool {
	// Resolve addresses
	serverAddr, err := net.ResolveUDPAddr("udp", common.ServerIP+":"+common.ServerPort)
	if err != nil {
		fmt.Println("Error resolving server address:", err)
		return false
	}

	clientAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%s", ownPort))
	if err != nil {
		fmt.Println("Error resolving client address:", err)
		return false
	}

	// Set up UDP connection for unicast communication
	conn, err := net.ListenUDP("udp", clientAddr)
	if err != nil {
		fmt.Println("Error setting up client connection:", err)
		return false
	}
	defer conn.Close()

	// Send ready check to the server
	_, err = conn.WriteToUDP([]byte("READY"), serverAddr)
	if err != nil {
		fmt.Println("Error sending ready check:", err)
		return false
	}

	// Join multicast group to receive video stream
	multicastAddr, err := net.ResolveUDPAddr("udp", common.MulticastIP+":"+port)
	if err != nil {
		fmt.Println("Error resolving multicast address:", err)
		return false
	}

	mconn, err := net.ListenMulticastUDP("udp", nil, multicastAddr)
	if err != nil {
		fmt.Println("Error joining multicast group:", err)
		return false
	}
	defer mconn.Close()

	mconn.SetReadBuffer(1024)

	buf := make([]byte, 1024)
	for {
		n, src, err := mconn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving multicast message:", err)
			continue
		}
		if string(buf[:n]) == "FINISHED" {
			fmt.Printf("Received message from %s\n", src)
			return true
		}
	}
}
