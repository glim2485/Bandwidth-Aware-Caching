package main

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/server"
	"gjlim2485/bandwidthawarecaching/user"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"runtime/pprof"
	"strconv"
	"sync"
	"time"

	"github.com/xuri/excelize/v2"
)

func main() {
	stopMemProfile := startProfile("memprofile_", 1*time.Second)
	defer close(stopMemProfile)

	go server.SimulStartServer()
	go server.SimulStartCloud()
	go garbageCollectionFunc(1 * time.Second)
	var wg sync.WaitGroup
	time.Sleep(5 * time.Second)
	startTime := time.Now()
	for i := 0; i < common.UserCount; i++ {
		wg.Add(1)
		go user.SimulUserRequests(i, common.UserIterations, common.UserCacheSize, &wg)
	}
	wg.Wait()
	endTime := time.Now()
	fmt.Println("simulation finished, now logging into excel sheet")
	//log data to excel sheet
	f, err := excelize.OpenFile("/home/dnclab/Bandwidth-Aware-Caching/dataLog.xlsx")
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
	sheetName := fmt.Sprintf("%d_%d_%d %d_%d", year, month, day, hour, minute)

	index, err := f.NewSheet(sheetName)
	if err != nil {
		fmt.Println(err)
	}
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
	fmt.Println("sucessfully logged to dataLog.xlsx")
}

// debugging functions
func startProfile(filenamePrefix string, interval time.Duration) chan bool {
	stop := make(chan bool)
	go func() {
		currDir := "/home/dnclab/Bandwidth-Aware-Caching/log"
		latestFolder, _ := findLatestFolderNumber(currDir)
		folderName := fmt.Sprintf("log_%d", latestFolder+1)
		logDir := filepath.Join(currDir, folderName)
		os.Mkdir(logDir, 0755)
		memDir := filepath.Join(logDir, "memLog")
		cpuDir := filepath.Join(logDir, "cpuLog")
		readMemDir := filepath.Join(logDir, "readMemLog")
		os.Mkdir(memDir, 0755)
		os.Mkdir(cpuDir, 0755)
		os.Mkdir(readMemDir, 0755)
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		count := 0
		var m runtime.MemStats
		readMemFile := fmt.Sprintf("%s.log", "readMemory")
		f3, _ := os.Create(filepath.Join(readMemDir, readMemFile))
		defer f3.Close()
		for {
			select {
			case <-ticker.C:
				memFile := fmt.Sprintf("%s%d.out", "memory_", count)
				cpuFile := fmt.Sprintf("%s%d.out", "cpu_", count)

				f1, _ := os.Create(filepath.Join(memDir, memFile))
				defer f1.Close()
				f2, _ := os.Create(filepath.Join(cpuDir, cpuFile))
				if err := pprof.Lookup("heap").WriteTo(f1, 0); err != nil {
					log.Printf("could not write memory profile %s: %v", memFile, err)
				}

				if err := pprof.StartCPUProfile(f2); err != nil {
					log.Printf("could not write cpu profile %s: %v", cpuFile, err)
				} else {
					time.Sleep(interval)
					pprof.StopCPUProfile()
				}
				f2.Close()
				runtime.ReadMemStats(&m)
				logger := log.New(f3, "", log.LstdFlags)
				logger.Printf("Alloc = %v MiB", bToMb(m.Alloc))
				logger.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
				logger.Printf("\tSys = %v MiB", bToMb(m.Sys))
				logger.Printf("\tHeapAlloc = %v MiB", bToMb(m.HeapAlloc))
				logger.Printf("\tHeapSys = %v MiB", bToMb(m.HeapSys))
				logger.Printf("\tHeapIdle = %v MiB", bToMb(m.HeapIdle))
				logger.Printf("\tHeapInuse = %v MiB", bToMb(m.HeapInuse))
				logger.Printf("\tHeapReleased = %v MiB", bToMb(m.HeapReleased))
				logger.Printf("\tHeapObjects = %v\n", m.HeapObjects)
				logger.Printf("\tNumGC = %v\n", m.NumGC)
				count++
			case <-stop:
				return
			}
		}
	}()
	return stop
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func findLatestFolderNumber(root string) (int, error) {
	latestNumber := 0
	re := regexp.MustCompile(`^log_(\d+)$`)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			matches := re.FindStringSubmatch(info.Name())
			if matches != nil {
				num, err := strconv.Atoi(matches[1])
				if err != nil {
					return err
				}
				if num > latestNumber {
					latestNumber = num
				}
			}
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return latestNumber, nil
}

func garbageCollectionFunc(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for range ticker.C {
		runtime.GC()
	}
}
