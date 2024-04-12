package apicalls

import (
	"gjlim2485/bandwidthawarecaching/common"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func SendData(c *gin.Context) {
	var userData common.UserData
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if !common.ToggleCoding {
		filePath := common.DataDirectory + "/" + userData.RequestData
		c.File(filePath)
	} else {
		if common.GlobalTimer.Stop() {
			common.GlobalTimer.Reset(common.TimerTime * time.Second)
		}
		common.UserDataChannel <- userData
		<-common.GlobalTimer.C
	}
}
