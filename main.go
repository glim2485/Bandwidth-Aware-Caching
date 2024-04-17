package main

import (
	"gjlim2485/bandwidthawarecaching/user"
)

func main() {
	common.NumberUsers := 100
	for i := 0; i < numberUsers; i++ {
		user.CreateUserThread(i)
	}
}
