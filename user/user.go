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
	"time"
)

type serverResponse struct {
	UserPort      string `json:"UserPort"`
	ServerPort    string `json:"ServerPort"`
	StatusMessage string `json:"StatusMessage"`
}

func SimulUserRequests(userid int, iteration int, cacheSize int, wg *sync.WaitGroup) {
	fmt.Println("User", userid, "started")
	//defer wg.Done()
	//set its port to be 40000 + userid
	ownPort := strconv.Itoa(40000 + userid)
	//all user cache inuse is set to false for experiment's sake
	userCache := lru.Constructor(cacheSize)
	for i := 0; i < iteration; i++ {
		userFiles := userCache.GetCacheList()
		userRequest := generateRequestFile()
		fmt.Println("User", userid, "iteration", i, "started for", userRequest)
		requestMessage := common.UserRequest{
			UserID:      userid,
			RequestFile: userRequest,
			UserFile:    userFiles,
		}
		//debug
		//fmt.Println("User", userid, "cache list:", userFiles, "and size is", unsafe.Sizeof(userFiles), "bytes")
		//fmt.Println("User", userid, "cache size is", unsafe.Sizeof(userCache), "bytes")
		//debug
		userFiles = nil
		jsonData, err := json.Marshal(requestMessage)
		if err != nil {
			fmt.Println("Error marshalling JSON:", err)
			return
		}
		startTime := time.Now()
		url := "http://" + common.ServerIP + ":" + common.ServerPort + "/getdata"
		body := bytes.NewBuffer(jsonData)
		//fmt.Println("User", userid, "requesting", userRequest, "from server")
		resp, err := http.Post(url, "application/json", body)
		if err != nil {
			fmt.Println("Error sending request:", err)
			return
		}
		defer resp.Body.Close()
		//fmt.Println(common.FetchType[resp.StatusCode])
		switch resp.StatusCode {
		case 200:
			userCache.Put(userRequest, 0)
		case 333:
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				//fmt.Println("Error reading response body:", err)
				return
			}
			var response serverResponse
			err = json.Unmarshal(body, &response)
			fmt.Println("user", userid, "received multicast request for", userRequest, "from port", response.UserPort, "to port", response.ServerPort)
			if err != nil {
				//fmt.Println("Error unmarshalling JSON:", err)
				return
			}
			if joinMulticast(response.UserPort, response.ServerPort, ownPort, userid, userRequest) {
				userCache.Put(userRequest, 0)
			} else {
				//fmt.Println("Error joining multicast group")
			}
		case 334:
			cloudUrl := "http://" + common.ServerIP + ":" + common.CloudPort + "/getdata"
			body := bytes.NewBuffer(jsonData)
			resp, err := http.Post(cloudUrl, "application/json", body)
			if err != nil {
				fmt.Println("Error sending request:", err)
				return
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				fmt.Println("Cloud fetch successful")
			}
		}
		totalTime := int(time.Since(startTime) / time.Millisecond)
		common.UserDataLogLock.Lock()
		common.UserDataLog = append(common.UserDataLog, common.UserDataLogStruct{
			UserID:      userid,
			RequestFile: userRequest,
			ReturnCode:  resp.StatusCode,
			FetchType:   common.FetchType[resp.StatusCode],
			TimeTaken:   totalTime,
		})
		common.UserDataLogLock.Unlock()
		//fmt.Println("User", userid, " current cache size:", userCache.GetLength())
	}
	fmt.Println("User", userid, "finished")
	wg.Done()
}

// case 335: was swapped with swapped item
// case 336: cache needs to be fetched from cloud, in-transit
func generateRequestFile() string {
	return "file" + strconv.Itoa(rand.Intn(50)+1)
}

func joinMulticast(userPort string, serverPort string, ownPort string, userid int, requestFile string) bool {
	// Resolve addresses
	serverAddr, err := net.ResolveUDPAddr("udp", common.ServerIP+":"+serverPort)
	if err != nil {
		fmt.Println("Error resolving server address:", err)
		return false
	}

	// Join multicast group to receive video stream
	multicastAddr, err := net.ResolveUDPAddr("udp", common.MulticastIP+":"+userPort)
	if err != nil {
		fmt.Println("Error resolving multicast address:", err)
		return false
	}

	//ready multicast tunnel
	mconn, err := net.ListenMulticastUDP("udp", nil, multicastAddr)
	if err != nil {
		fmt.Println("Error joining multicast group:", err)
		return false
	}
	defer mconn.Close()

	mconn.SetReadBuffer(1024)

	buf := make([]byte, 1024)

	//multicast reception ready, send ready message
	clientAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:"+ownPort)
	if err != nil {
		fmt.Println("Error resolving client address:", err)
		return false
	}
	conn, err := net.ListenUDP("udp", clientAddr)
	if err != nil {
		fmt.Println("Error setting up client connection:", err)
		return false
	}
	defer conn.Close()

	// Send ready check to the server
	closeChan := make(chan bool)
	go sendReadyMessage(userid, requestFile, serverPort, conn, serverAddr, closeChan) //send ready check 3 times
	//finish receiving multicast message
	for {
		n, _, err := mconn.ReadFromUDP(buf)
		//n, src, err := mconn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving multicast message:", err)
			continue
		}
		if string(buf[:n]) == "FINISHED" {
			//fmt.Printf("Received message from %s\n", src)
			closeChan <- true
			return true
		}
	}
}

func sendReadyMessage(userid int, requestFile string, serverPort string, conn *net.UDPConn, serverAddr *net.UDPAddr, closeChan <-chan bool) {
	ticker := time.NewTicker(2 * time.Second)
	for {
		select {
		case <-closeChan:
			return
		case <-ticker.C:
			_, err := conn.WriteToUDP([]byte("READY"+strconv.Itoa(userid)), serverAddr)
			if err != nil {
				fmt.Println("Error sending ready check:", err)
			} else {
				fmt.Println("User", userid, "sent ready check for file ", requestFile, "to port", serverPort)
			}
		case <-time.After(7 * time.Second):
			fmt.Println("User", userid, "timed out")
		}

	}
}
