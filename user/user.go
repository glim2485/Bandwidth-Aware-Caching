package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/data"
	"gjlim2485/bandwidthawarecaching/server"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
)

func CreateUserThread(wg *sync.WaitGroup, userID int) {
	defer wg.Done()
	zipf := data.GetZipfDistribution(int64(userID)) //generate its own zipf distribution
	cachedItems := common.Constructor(common.MaxLocalCacheSize)
	var logInput common.UserCacheHit
	for i := 0; i < common.UserIteration; i++ {
		requestData := zipf.Uint64() + 1 //to make it [0,100]
		filename := "data" + strconv.Itoa(int(requestData)) + ".mp4"

		//Check local/user cache first before sending request to edge
		checkLocalCache := cachedItems.Get(filename)
		if checkLocalCache != "" {
			logInput = common.UserCacheHit{ItemName: filename, CacheHit: "local", TimeTaken: 0, Multicast: false}
		} else {
			//If not in local cache, check edge cache
			localCache := cachedItems.GetCacheList()
			hit, size, connModifier := server.SimulIncomingData(userID, filename, localCache)
			var cacheHitLocation string
			if hit {
				cacheHitLocation = "edge"
			} else {
				cacheHitLocation = "cloud"
			}

			var didMulticast bool
			if common.ToggleMulticast {
				didMulticast = true
			} else {
				didMulticast = false
			}
			timeTaken := common.SimulTransferringData(size)
			common.SimulUpdateConcurrentConnection(-connModifier) //close connection
			common.EdgeCache.ChangeFileStatus(false, filename)
			cachedItems.Put(filename, filename)
			logInput = common.UserCacheHit{ItemName: filename, CacheHit: cacheHitLocation, TimeTaken: timeTaken, Multicast: didMulticast}
		}

		//add to log
		temp := common.UserLogInfo[userID]
		temp.UserResult = append(temp.UserResult, logInput)
		common.UserLogInfo[userID] = temp
	}
}

func RequestFile(filename string) {
	url := "testurl"
	userData := common.UserData{UserIP: common.MyIP, LocalCache: []string{"some_items.mp4"}, RequestData: filename}
	jsonData, _ := json.Marshal(userData)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error:", resp.Status)
		return
	}

	file, err := os.Create(common.DataDirectory + "/" + filename)
	if err != nil {
		fmt.Println("Error creating file:", err)
		return
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		fmt.Println("Error saving file:", err)
		return
	}

	fmt.Println("MP4 file received and saved successfully.")
}
