package main

import (
	"gjlim2485/bandwidthawarecaching/common"
	"gjlim2485/bandwidthawarecaching/server"
	"gjlim2485/bandwidthawarecaching/user"
	"sync"
)

func main() {
	server.SimulInitializeServer()
	var wg sync.WaitGroup
	common.UserNumbers = 100
	wg.Add(common.UserNumbers)
	for i := 0; i < common.UserNumbers; i++ {
		go user.CreateUserThread(&wg, i)
	}
	wg.Wait()
}
