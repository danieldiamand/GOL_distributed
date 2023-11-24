package stubs

var WorkerInit = "Worker.Init"
var WorkerStart = "Worker.Start"
var WorkerProgress = "Worker.Progress"
var WorkerHalo = "Worker.Halo"
var WorkerCount = "Worker.Count"
var WorkerFetch = "Worker.Fetch"
var WorkerKill = "Worker.Kill"

var BrokerQueryState = "Broker.QueryState"
var BrokerInit = "Broker.Init"
var BrokerStart = "Broker.Start"
var BrokerProgressAll = "Broker.ProgressAll"
var BrokerCount = "Broker.Count"
var BrokerPause = "Broker.Pause"
var BrokerFetch = "Broker.Fetch"
var BrokerQuit = "Broker.Quit"
var BrokerKill = "Broker.Kill"

var DistReceive = "Distributor.Receive"

type None struct {
	//Empty
}

type BrokerStateRes struct {
	StillCalculating bool
	Details          string
}

type BrokerInitReq struct {
	World         [][]byte
	Width         int
	Height        int
	Turns         int
	PrintProgress bool
}

type BrokerStartReq struct {
	WorkerCount     int
	WorkerAddresses []string
}

type WorkerInitReq struct {
	World         [][]byte
	Width         int
	Height        int
	PrintProgress bool
}

type WorkerStartReq struct {
	AboveAdr string
}

type Turn struct {
	Turn int
}

type WorkerHaloReqRes struct {
	Halo []byte
}

type WorldRes struct {
	World [][]byte
	Turn  int
}

type CountCellRes struct {
	Count int
	Turn  int
}

type PauseRes struct {
	Output string
}
