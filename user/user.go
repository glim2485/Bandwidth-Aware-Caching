package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gjlim2485/bandwidthawarecaching/codecache"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/data"
	"gjlim2485/bandwidthawarecaching/server"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

func CreateUserThread(wg *sync.WaitGroup, userID int) {
	defer wg.Done()
	zipf := data.GetZipfDistribution(int64(userID)) //generate its own zipf distribution
	cachedItems := codecache.Constructor(common.MaxLocalCacheSize)
	var logInput common.UserCacheHit
	for i := 0; i < common.UserIteration; i++ {
		requestData := zipf.Uint64() + 1 //to make it [0,100]
		filename := "data" + strconv.Itoa(int(requestData)) + ".mp4"

		//Check local/user cache first before sending request to edge
		checkLocalCache := cachedItems.Get(filename)
		if checkLocalCache != "" {
			logInput = common.UserCacheHit{ItemName: filename, CacheHit: "local", TimeTaken: 0}
		} else {
			//If not in local cache, check edge cache
			localCache := cachedItems.GetCacheList()
			hit, size := server.SimulIncomingData(userID, filename, localCache)
			if hit {
				if common.ToggleMulticast {
					timeTaken := SimulTrasferringData(size)
					cachedItems.Put(filename, filename)
					logInput = common.UserCacheHit{ItemName: filename, CacheHit: "edge", TimeTaken: timeTaken}
				} else {

				}
			} else {
				//not in edge, need to request from cloud
				//TODO: develop request to cloud
			}
		}

		//add to log
		temp := common.UserLogInfo[userID]
		temp.UserResult = append(temp.UserResult, logInput)
		common.UserLogInfo[userID] = temp
	}
}

func SimulTrasferringData(filesize int) int {
	transferred_data := 0
	currentTime := time.Now()
	for transferred_data < filesize {
		transferred_data += int(common.SplitBandwidth)
		time.Sleep(1 * time.Second)
	}
	newTime := time.Now()
	timeTaken := newTime.Sub(currentTime)
	timeTakenInt := int(timeTaken.Milliseconds())
	return timeTakenInt
}

func RequestFile(filename string) {
	url := "testurl"
	userData := common.UserData{UserIP: common.MyIP, LocalCache: CachedItems, RequestData: filename}
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
