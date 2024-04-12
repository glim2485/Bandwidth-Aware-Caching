package server

import (
	"fmt"
	"gjlim2485/bandwidthawarecaching/common"
	"net"

	"github.com/gin-gonic/gin"
)

func HTTPServer() {
	router := gin.Default()

	if err := router.Run(common.ServerPort); err != nil {
		fmt.Println("Error starting HTTP server")
		return
	}

	router.Run("0.0.0.0:8080")
}

func MulticastServer(portNumber int) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{
		IP:   net.ParseIP(common.MulticastIP),
		Port: portNumber,
	})
	if err != nil {
		fmt.Println("Error creating UDP connection:", err)
		return
	}
	defer conn.Close()
}

func CollectData() {
	var dataCollection []common.UserData
	for {
		select {
		case <-common.GlobalTimer.C:
			if len(dataCollection) != 0 {
				//check for relations
			}
			dataCollection = nil //reset the
		default:
			incomingData := <-common.UserDataChannel
			dataCollection = append(dataCollection, incomingData)
		}
	}
}
