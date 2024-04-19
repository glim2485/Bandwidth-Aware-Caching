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
	server.SimulInitializeServer()
	var wg sync.WaitGroup
	common.UserNumbers = 100
	wg.Add(common.UserNumbers)
	startTime := time.Now()
	for i := 0; i < common.UserNumbers; i++ {
		go user.CreateUserThread(&wg, i)
	}
	wg.Wait()

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
	t := time.Now()
	totalTime := t.Sub(startTime)
	year, month, day := t.Date()
	hour, minute := t.Hour(), t.Minute()
	sheetName := fmt.Sprintf("%d-%d-%d %d:%d", year, month, day, hour, minute)

	index, err := f.NewSheet(sheetName)
	f.SetCellValue(sheetName, "A1", "UserID")
	f.SetCellValue(sheetName, "B1", "ItemName")
	f.SetCellValue(sheetName, "C1", "CacheHit")
	f.SetCellValue(sheetName, "D1", "TimeTaken")
	cellIndex := 1
	for i, d := range common.UserLogInfo {
		for _, y := range d.UserResult {
			cellIndex++
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", cellIndex), i)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", cellIndex), y.ItemName)
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", cellIndex), y.CacheHit)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", cellIndex), y.TimeTaken)
		}
	}
	cellIndex++
	totalDurationMilliseconds := int64(totalTime / time.Millisecond)
	f.SetCellValue(sheetName, fmt.Sprintf("A%d", cellIndex), "Total Duration")
	f.SetCellValue(sheetName, fmt.Sprintf("B%d", cellIndex), fmt.Sprintf("%d ms", totalDurationMilliseconds))
	f.SetActiveSheet(index)
	if err = f.Save(); err != nil {
		fmt.Println(err)
	}
}
