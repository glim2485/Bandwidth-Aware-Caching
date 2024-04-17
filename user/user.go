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
	"time"
)

func CreateUserThread(userID int) {
	zipf := data.GetZipfDistribution(int64(userID)) //generate its own zipf distribution
	var CachedItems []string
	for i := 0; i < common.UserIteration; i++ {
		requestData := zipf.Uint64() + 1 //to make it [0,100]
		filename := "data" + strconv.Itoa(int(requestData)) + ".mp4"

		if SimulRequestData(filename) {
			CachedItems = append(CachedItems, filename)
		}
	}
}

func SimulRequestData(filename string) bool {
	request_file := "somedata.mp4"
	hit, size := server.SimulIncomingData(request_file)
	if hit {
		if SimulTrasferringData(size) {
			return true
		}
	}
	return false
}

func SimulTrasferringData(filesize int) bool {
	transferred_data := 0
	for transferred_data < filesize {
		transferred_data += int(common.SplitBandwidth)
		time.Sleep(1 * time.Second)
	}
	return true
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
