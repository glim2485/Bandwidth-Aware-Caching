package main

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/server"
	"gjlim2485/bandwidthawarecaching/user"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

func main() {
	go server.SimulStartServer()
	go server.SimulStartCloud()
	var wg sync.WaitGroup
	time.Sleep(5 * time.Second)
	startTime := time.Now()
	for i := 0; i < common.UserCount; i++ {
		wg.Add(1)
		user.SimulUserRequests(i+1, common.UserIterations, common.UserCacheSize, &wg)
	}
	wg.Wait()
	endTime := time.Now()
	//log data to excel sheet
	f, err := excelize.OpenFile("dataLog.xlsx")
	if err != nil {
		fmt.Println(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()
	totalTime := endTime.Sub(startTime)
	year, month, day := startTime.Date()
	hour, minute := startTime.Hour(), startTime.Minute()
	sheetName := fmt.Sprintf("%d-%d-%d %d:%d", year, month, day, hour, minute)

	index, _ := f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "UserCacheSize")
	f.SetCellValue(sheetName, "B1", fmt.Sprintf("%d", common.UserCacheSize))
	f.SetCellValue(sheetName, "A2", "edgeCacheSize")
	f.SetCellValue(sheetName, "B2", fmt.Sprintf("%d", common.EdgeCacheSize))
	f.SetCellValue(sheetName, "A3", "EdgeBandwidth")
	f.SetCellValue(sheetName, "B3", fmt.Sprintf("%.4f", common.MaxBandwidth))
	f.SetCellValue(sheetName, "A4", "SwapItemSize")
	f.SetCellValue(sheetName, "B4", fmt.Sprintf("%d", common.SwapItemSize))

	f.SetCellValue(sheetName, "A6", "UserID")
	f.SetCellValue(sheetName, "B6", "ItemName")
	f.SetCellValue(sheetName, "C6", "CacheHit")
	f.SetCellValue(sheetName, "D6", "TimeTaken(ms)")
	cellIndex := 6
	fetchCount := make(map[int]int)
	for _, d := range common.UserDataLog {
		cellIndex++
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", cellIndex), d.UserID)
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", cellIndex), d.RequestFile)
		f.SetCellValue(sheetName, fmt.Sprintf("C%d", cellIndex), d.FetchType)
		f.SetCellValue(sheetName, fmt.Sprintf("D%d", cellIndex), d.TimeTaken)
		fetchCount[d.ReturnCode]++
	}
	cellIndex++
	totalDurationMilliseconds := int64(totalTime / time.Millisecond)
	f.SetCellValue(sheetName, "F1", "Total Duration")
	f.SetCellValue(sheetName, "G1", fmt.Sprintf("%d ms", totalDurationMilliseconds))
	f.SetCellValue(sheetName, "F3", "HTTP code")
	f.SetCellValue(sheetName, "G3", "Fetch Type")
	f.SetCellValue(sheetName, "H3", "Count")

	fetchCellValue := 4
	for k, v := range common.FetchType {
		f.SetCellValue(sheetName, fmt.Sprintf("F%d", fetchCellValue), k)
		f.SetCellValue(sheetName, fmt.Sprintf("G%d", fetchCellValue), v)
		f.SetCellValue(sheetName, fmt.Sprintf("H%d", fetchCellValue), fetchCount[k])
		fetchCellValue++
	}
	f.SetActiveSheet(index)
	if err = f.Save(); err != nil {
		fmt.Println(err)
	}
}
