package server

func Start() {
	Accept()
	ListenTokens()
	RecheckIota()
	ListenTxErrorRecord()
}

func Stop() {
}
