package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"net"
	"os"
	"sync"
	"time"
)

var collectedData = make([]common.UserRequest, 0)
var currUDPPort = 40000
var UDPlock sync.RWMutex
var multicastWaitTime = time.Duration(3) * time.Second

type multicastGroup struct {
	userID        []int
	intersection  []string
	multicastPort string
}

func progressUDPPort() {
	UDPlock.Lock()
	defer UDPlock.Unlock()
	currUDPPort++
	if currUDPPort >= 60001 {
		currUDPPort = 50000
	}
}

func handleData(userData []common.UserRequest, port int) {
	//find multicast targets
	multicastGroups := MulticastGroup(userData)

	//if no code cache is needed
	if !common.EnableMulticast {
		for filename, x := range multicastGroups {
			go multicastData(x.userID, filename, x.multicastPort)
		}
		return
	}
	//do code cache if necessary
	//send data in multicast
}

func MulticastGroup(userData []common.UserRequest) map[string]multicastGroup {
	returnGroup := make(map[string]multicastGroup)

	//no need to find intersection if code cache is disabled
	//time complexity O(n)
	if !common.EnableCodeCache {
		for _, x := range userData {
			if group, exists := returnGroup[x.RequestFile]; exists {
				group.userID = append(group.userID, x.UserID)
				returnGroup[x.RequestFile] = group
			} else {
				progressUDPPort()
				returnGroup[x.RequestFile] = multicastGroup{
					userID:        []int{x.UserID},
					multicastPort: fmt.Sprintf("%d", currUDPPort),
				}
			}
		}
	} else {
		//code cache enabled so intersections are needed
		// group them all by request file and do find their intersection
		for _, x := range userData {
			if group, exists := returnGroup[x.RequestFile]; exists {
				newIntersection := getIntersection(group.intersection, x.UserFile)
				group.userID = append(group.userID, x.UserID)
				group.intersection = newIntersection
				returnGroup[x.RequestFile] = group
			} else {
				//do not make a UDP port yet
				returnGroup[x.RequestFile] = multicastGroup{
					userID:       []int{x.UserID},
					intersection: x.UserFile,
				}
			}
		}
	}
	return returnGroup
}

func getIntersection(setA []string, setB []string) []string {
	// O(n+m) time complexity
	if len(setA) > len(setB) {
		setA, setB = setB, setA
	}

	hash := make(map[string]bool)
	for _, x := range setA {
		hash[x] = true
	}

	var result []string
	for _, y := range setB {
		if _, exists := hash[y]; exists {
			result = append(result, y)
		}
	}
	return result
}

// made differently due to simulator
func multicastData(users []int, file string, port string) {
	//spread port for users
	for _, x := range users {
		udpAnnounceChannel[x] <- port
	}

	readyChan := make(chan string, len(users))
	var wg sync.WaitGroup
	serverAddr, err := net.ResolveUDPAddr("udp", common.ServerIP+":"+common.ServerPort)
	if err != nil {
		fmt.Println("error creating multicast address")
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", serverAddr)
	if err != nil {
		fmt.Println("Error listening to UDP:", err)
		os.Exit(1)
	}
	defer conn.Close()

	//check for all connections
	for range users {
		wg.Add(1)
		go receiveReadyCheck(conn, readyChan, &wg)
	}

	go func() {
		wg.Wait()
		close(readyChan)
	}()

	allReady := true
	for range users {
		select {
		case <-readyChan:
			fmt.Println("Ready check received")
		case <-time.After(10 * time.Second):
			allReady = false
		}
	}

	if allReady {
		fmt.Println("All users ready, multicasting ", file)
	} else {
		fmt.Println("Not all users ready")
		os.Exit(1)
	}

	//multicast the data now
	multicastAddr := common.MulticastIP + ":" + port
	addr, err := net.ResolveUDPAddr("udp", multicastAddr)
	if err != nil {
		fmt.Println("error creating multicast address")
		os.Exit(1)
	}
	multconn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		fmt.Println("Error dialing UDP:", err)
		os.Exit(1)
	}
	defer multconn.Close()
	//here, we should be sending data, but instead we will simulate it
	simulSendData(1)
	simulmsg := "FINISHED"
	_, err = conn.Write([]byte(simulmsg))
	if err != nil {
		fmt.Println("Error sending message:", err)
		os.Exit(1)
	}
	fmt.Println("Multicast transmission finished")
}

func receiveReadyCheck(conn *net.UDPConn, readyChan chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	n, addr, err := conn.ReadFromUDP(buf)
	if err != nil {
		fmt.Println("Error reading from UDP:", err)
		os.Exit(1)
	}
	if string(buf[:n]) == "READY" {
		readyChan <- addr.String()
	}
}
