package user

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"io"
	"net/http"
	"os"
)

var CachedItems []string

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
