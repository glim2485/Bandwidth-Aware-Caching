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
var multicastWaitTime = time.Duration(common.MulticastCollectTime) * time.Second

type multicastGroup struct {
	userID              []int
	intersection        []string
	multicastPortUser   string
	multicastPortServer string
}

func progressUDPPort() int {
	UDPlock.Lock()
	defer UDPlock.Unlock()
	currUDPPort++
	if currUDPPort >= 60000 {
		currUDPPort = 50000
	}
	return currUDPPort
}

func handleData(userData []common.UserRequest, port int) {
	//find multicast targets
	//fmt.Println("finding multicast groups for", userData)
	multicastGroups := MulticastGroup(userData)
	var serviceGroup map[string]multicastGroup
	if !common.EnableCodeCache {
		//code cache disabled, just use multicast group
		serviceGroup = multicastGroups
	} else {
		//fmt.Println("finding codecache groups for", multicastGroups)
		serviceGroup = codeCacheGroup(multicastGroups)
		//fmt.Println("codecache found", serviceGroup)
	}

	//fmt.Println("Multicasting/Codecaching", serviceGroup)
	for filename, x := range serviceGroup {
		copiedGroup := make([]int, len(x.userID))
		//to avoid changing midway?
		copy(copiedGroup, x.userID)
		go multicastData(copiedGroup, filename, x.multicastPortUser, x.multicastPortServer)
		//if no code cache is needed
		//TODO: missing code cache condition
	}
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
				multicastUser := fmt.Sprintf("%d", progressUDPPort())
				multicastServer := fmt.Sprintf("%d", progressUDPPort())
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

func codeCacheGroup(groups map[string]multicastGroup) map[string]multicastGroup {
	returnGroup := make(map[string]multicastGroup)
	deleteGroup := make([]string, 0)

	for filename, group := range groups {
		//check if file is in cache. If it is, add a counter
		edgeCacheLock.Lock()
		exists, _ := edgeCache.Get(filename, 1)
		edgeCacheLock.Unlock()
		if !exists {
			//if it doesnt exist in cache, we can't code cache it so multicast this separately
			//we store all the delete keys to delete later to avoid memory issues
			deleteGroup = append(deleteGroup, filename)
			//add this group to the return group since it wont be modified
			returnGroup[filename] = group
		}
	}

	//get rid of all the non-code cacheable groups
	for _, filename := range deleteGroup {
		delete(groups, filename)
	}

	//from here on, all members of groups are in the cache and we can start matching patterns
	codedGroup := groups
	for i := 0; i < common.MaxCodedItems; i++ {
		//swap group used to dynamically update codedGroups
		swapGroup := make(map[string]multicastGroup)
		//codedVisit is used to avoid double counting
		codedMerge := make(map[string]bool)
		deleteGroup := []string{}
		for filename1, x := range codedGroup {
			if codedMerge[filename1] {
				continue
			}
			for filename2, y := range codedGroup {
				if filename1 == filename2 || codedMerge[filename2] {
					continue
				}
				if sliceContainsFilenames(x.intersection, filename2) && sliceContainsFilenames(y.intersection, filename1) {
					//found a match, create a new group
					newIntersection := getIntersection(x.intersection, y.intersection)
					newGroup := multicastGroup{
						userID:       append(x.userID, y.userID...),
						intersection: newIntersection,
					}
					//these two can be deleted
					deleteGroup = append(deleteGroup, filename1, filename2)
					swapGroup[filename1+"_"+filename2] = newGroup
					codedMerge[filename1] = true
					codedMerge[filename2] = true
					//dont code any further for filename1
					break
				}
			}
		}
		if len(swapGroup) == 0 {
			//swapGroup having 0 length means no new groups were created
			break
		}
		//delete items that were grouped
		for _, x := range deleteGroup {
			delete(codedGroup, x)
		}
		//insert the new mixed items into the codedGroup for the next iteration
		for filename, group := range swapGroup {
			codedGroup[filename] = group
		}
	}

	//add ports before returning
	for filename, group := range codedGroup {
		multicastUser := fmt.Sprintf("%d", progressUDPPort())
		multicastServer := fmt.Sprintf("%d", progressUDPPort())
		group.multicastPortUser = multicastUser
		group.multicastPortServer = multicastServer
		returnGroup[filename] = group
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
	var serverAddr *net.UDPAddr
	var err error
	var conn *net.UDPConn

	for {
		serverAddr, err = net.ResolveUDPAddr("udp", common.ServerIP+":"+serverPort)
		if err != nil {
			fmt.Println("error creating multicast address for server",common.ServerIP,"port", serverPort)
			//get another udp port
			serverPort = fmt.Sprintf("%d", progressUDPPort())
			time.Sleep(2 * time.Second)
			continue
		}
		conn, err = net.ListenUDP("udp", serverAddr)
		if err != nil {
			fmt.Println("Error listening to UDP:", err)
			//get another udp port
			serverPort = fmt.Sprintf("%d", progressUDPPort())
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
	defer conn.Close()
	fmt.Println(file, "need ready from users:", users, "at port", serverPort)

	//spread port for users
	// userPort is the port the users are going to connect to RECEIVE data

	//debugging: I am creating a copy of users because for some reason, it does not always fully loop
	//to send the channel, it works fine on all subsequent logic checks for some reason and idk why
	//TODO: find the source of this bug
	copyUser := make([]int, len(users))
	copy(copyUser, users)
	for _, x := range copyUser {
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
			if !common.SliceContainsInt(readyUsers, ready) {
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
		fmt.Println("All users ready, multicasting ", file, " through port", userPort)
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
	simulmsg := "FINISHED_" + file
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

func sliceContainsFilenames(slice []string, filename string) bool {
	re := regexp.MustCompile("_")
	parts := re.Split(filename, -1)
	maxVote := len(parts)
	vote := 0
	for _, x := range parts {
		if common.SliceContainsString(slice, x) {
			vote++
		}
		if vote == maxVote {
			return true
		}
	}
	return false
}
