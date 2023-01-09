package main

import (
	"biota_swap/config"
	"biota_swap/daemon"
	"biota_swap/gl"
	"biota_swap/model"
	"biota_swap/server"
)

func main() {
	if config.Env != "debug" {
		daemon.Background("./out.log", true)
	}

	gl.CreateLogFiles()

	model.ConnectToMysql()

	server.ListenTokens()

	daemon.WaitForKill()
}
