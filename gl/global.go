package gl

import (
	"log"
	"os"
	"sync"

	"github.com/triplefi/go-logger/logger"
)

var (
	CleanupChan  = make(chan struct{})
	TopWaitGroup = new(sync.WaitGroup)
)

// WaitAndCleanup wait and cleanup
func WaitAndCleanup(doCleanup func()) {
	<-CleanupChan
	doCleanup()
}

var BeginBlockNumber uint64
var EndBlockNumber uint64
var PerBlockSize uint64 = 1000

// OutLogger global logger
var OutLogger *logger.Logger

func CreateLogFiles() {
	var err error
	if err = os.MkdirAll("./logs", os.ModePerm); err != nil {
		log.Panic("Create dir './logs' error. " + err.Error())
	}
	if OutLogger, err = logger.New("logs/out.log", 1, 3, 0); err != nil {
		log.Panic("Create Outlogger file error. " + err.Error())
	}
}
