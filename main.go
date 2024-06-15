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
	"flag"

	"github.com/xuri/excelize/v2"
)

func init() {
    flag.BoolVar(&common.EnableMulticast, "enableMulticast", true, "enable multicast feature")
    flag.BoolVar(&common.EnableCodeCache, "enableCodecast", true, "enable codecast feature")
    flag.Int64Var(&common.SeedMultiplier, "seedValue", 7, "seed value")
	flag.IntVar(&common.UserCount, "userCount", 100, "number of users")
	flag.IntVar(&common.UserIterations, "userIterations", 25, "number of iterations per user")
	flag.IntVar(&common.UserCacheSize, "userCacheSize", 25, "size of user cache")
	flag.Float64Var(&common.DataSize, "dataSize", 500, "size of data")
	flag.Float64Var(&common.MaxBandwidth, "maxBandwidth", 2500, "max bandwidth")
	flag.IntVar(&common.EdgeCacheSizeMultiplier, "edgeCacheSizeMultiplier", 10, "edge cache size multiplier")
	flag.IntVar(&common.MaxFiles, "maxFiles", 50, "max files")
}

func main() {
	flag.Parse()
	common.EdgeCacheSize = common.UserCacheSize * common.EdgeCacheSizeMultiplier
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
	excelDir, err := os.Getwd()
	excelDir = filepath.Join(excelDir, "dataLog.xlsx")
	f, err := excelize.OpenFile(excelDir)
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

	f.SetCellValue(sheetName, "D1", "Multicast")
	f.SetCellValue(sheetName, "E1", fmt.Sprintf("%t", common.EnableMulticast))
	f.SetCellValue(sheetName, "D2", "CodeCache")
	f.SetCellValue(sheetName, "E2", fmt.Sprintf("%t", common.EnableCodeCache))
	f.SetCellValue(sheetName, "D3", "MaxCodeLoop")
	f.SetCellValue(sheetName, "E3", fmt.Sprintf("%d", common.MaxCodedItems))
	f.SetCellValue(sheetName, "D4", "Seed")
	f.SetCellValue(sheetName, "E4", fmt.Sprintf("%d", common.SeedMultiplier))

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

	f.SetCellValue(sheetName, "G1", "MulticastMultiplier")
	f.SetCellValue(sheetName, "H1", fmt.Sprintf("%.1f", common.MulticastBandwidthMultiplier))
	f.SetCellValue(sheetName, "G2", "MulticastCollectTime")
	f.SetCellValue(sheetName, "H2", fmt.Sprintf("%d", common.MulticastCollectTime))

	totalDurationMilliseconds := int64(totalTime / time.Millisecond)
	f.SetCellValue(sheetName, "J1", "Total Duration")
	f.SetCellValue(sheetName, "K1", fmt.Sprintf("%d ms", totalDurationMilliseconds))
	f.SetCellValue(sheetName, "J2", "userNumber")
	f.SetCellValue(sheetName, "K2", fmt.Sprintf("%d", common.UserCount))
	f.SetCellValue(sheetName, "J3", "Iteration Count")
	f.SetCellValue(sheetName, "K3", fmt.Sprintf("%d", common.UserIterations))

	f.SetCellValue(sheetName, "J5", "HTTP code")
	f.SetCellValue(sheetName, "K5", "Fetch Type")
	f.SetCellValue(sheetName, "I5", "Count")

	fetchCellValue := 6
	for k, v := range common.FetchType {
		f.SetCellValue(sheetName, fmt.Sprintf("J%d", fetchCellValue), k)
		f.SetCellValue(sheetName, fmt.Sprintf("K%d", fetchCellValue), v)
		f.SetCellValue(sheetName, fmt.Sprintf("L%d", fetchCellValue), fetchCount[k])
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