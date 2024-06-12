package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"
)

var collectedData = make([]common.UserRequest, 0)
var currUDPPort = 50000
var UDPlock sync.RWMutex
var multicastWaitTime = time.Duration(2) * time.Second

type multicastGroup struct {
	userID              []int
	intersection        []string
	multicastPortUser   string
	multicastPortServer string
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
	fmt.Println("finding multicast groups for", userData)
	multicastGroups := MulticastGroup(userData)

	//if no code cache is needed
	if !common.EnableCodeCache {
		for filename, x := range multicastGroups {
			copiedGroup := make([]int, len(x.userID))
			//to avoid changing midway?
			copy(copiedGroup, x.userID)
			go multicastData(copiedGroup, filename, x.multicastPortUser, x.multicastPortServer)
		}
	} else {

	}
	//TODO: missing code cache condition
}

func codeCacheGroup(groups map[string]multicastGroup) map[string]multicastGroup {
	//groups contain all the multicast groups along with their intersection
	//time complexity O(n^2)
	returnGroup := make(map[string]multicastGroup)
	for filename, _ := range groups {
		edgeCacheLock.Lock()
		exists, _ := edgeCache.Get(filename, 1)
		edgeCacheLock.Unlock()
		//if it does not exist in cache, we can't code cache it so multicast this separately
		if !exists {
			returnGroup[filename] = groups[filename]
			delete(groups, filename)
		}
	}
	//from here on, all members of groups are in the cache and we can start matching patterns
	for filename, x := range groups {
		//TODO: need to think of a way to match code patterns
	}
	return returnGroup
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
				multicastUser := fmt.Sprintf("%d", currUDPPort)
				progressUDPPort()
				multicastServer := fmt.Sprintf("%d", currUDPPort)
				returnGroup[x.RequestFile] = multicastGroup{
					userID:              []int{x.UserID},
					multicastPortUser:   multicastUser,
					multicastPortServer: multicastServer,
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
func multicastData(users []int, file string, userPort string, serverPort string) {
	fmt.Println("Server preparing to multicast", file, "to users", users, "through port", serverPort)
	readyChan := make(chan int, len(users))
	var wg sync.WaitGroup
	//serverPort is used for the server to receive data
	serverAddr, err := net.ResolveUDPAddr("udp", common.ServerIP+":"+serverPort)
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
	fmt.Println(file, "need ready from users:", users, "at port", serverPort)

	//spread port for users
	// userPort is the port the users are going to connect to RECEIVE data
	for _, x := range users {
		//channels should have been created by now
		udpAnnounceChannel[x] <- [2]string{userPort, serverPort}
		fmt.Println("Server sent udpAnnounceChannel", serverPort, "to user", x, "out of", users, "for", file)
	}

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
	currCount := 0
	timeout := time.After(20 * time.Second)
	readyUsers := []int{}
	for {
		select {
		case ready := <-readyChan:
			if !sliceContainsInt(readyUsers, ready) {
				currCount++
				fmt.Println("Ready check received in port", serverPort, "for", file, "from user", ready, "(", currCount, "/", len(users), ")")
				readyUsers = append(readyUsers, ready)
				if currCount == len(users) {
					allReady = true
					break
				}
			}
		case <-timeout:
			allReady = false
			break
		}
		if currCount == len(users) || !allReady {
			break
		}
	}

	if allReady {
		fmt.Println("All users ready, multicasting ", file, " through port", serverPort)
	} else {
		fmt.Println("Not all users ready for file", file, "ready:", readyUsers, "expected:", users, "from port", serverPort)
		os.Exit(1)
	}

	//multicast the data now
	multicastAddr := common.MulticastIP + ":" + userPort
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
	//error happens here
	_, err = multconn.Write([]byte(simulmsg))
	if err != nil {
		fmt.Println("Error sending message:", err)
		os.Exit(1)
	}
	fmt.Println("Multicast transmission finished for", file, "to users", users, ". Closing port", serverPort)
}

func receiveReadyCheck(conn *net.UDPConn, readyChan chan<- int, wg *sync.WaitGroup) {
	defer wg.Done()
	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(20 * time.Second))
	n, _, err := conn.ReadFromUDP(buf)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("Read timeout:", err)
			return
		}
		fmt.Println("Error reading from UDP:", err)
		return
	}
	status, returnID := readyRegex(string(buf[:n]))
	if status == "READY" {
		readyChan <- returnID
	}
}

func readyRegex(input string) (string, int) {
	re := regexp.MustCompile(`^(?P<ready>READY)(?P<userid>\d+)$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) != 3 {
		fmt.Println("Invalid ready message")
		return "false", 0
	}
	userID, err := strconv.Atoi(matches[2])
	if err != nil {
		return "false", 0
	}
	return matches[1], userID
}

func sliceContainsInt(slice []int, item int) bool {
	for _, x := range slice {
		if x == item {
			return true
		}
	}
	return false
}

func sliceContainsString(slice []string, item string) bool {
	for _, x := range slice {
		if x == item {
			return true
		}
	}
	return false
}
