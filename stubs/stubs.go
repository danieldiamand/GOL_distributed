package stubs

var WorkerInit = "Worker.Init"
var WorkerStart = "Worker.Start"
var WorkerProgress = "Worker.Progress"
var WorkerHalo = "Worker.Halo"
var WorkerCount = "Worker.Count"
var WorkerFetch = "Worker.Fetch"
var WorkerQuit = "Worker.Quit"
var WorkerKill = "Worker.Kill"

var BrokerProgressWorld = "Broker.ProgressWorld"
var BrokerCount = "Broker.Count"
var BrokerPause = "Broker.Pause"
var BrokerFetchWorld = "Broker.Fetch"
var BrokerQuit = "Broker.Quit"
var BrokerKill = "Broker.Kill"

const (
	Success                int = 0
	MoreThreadsThanWorkers     = 1
	CannotConnectToWorker      = 2
)

type None struct {
	//Empty
}

type BrokerProgressWorldReq struct {
	WorkersAdr    []string
	World         [][]byte
	Width         int
	Height        int
	Turns         int
	PrintProgress bool
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
	World     [][]byte
	Turn      int
	ErrorCode int
	ErrorMsg  string
}

type CountCellRes struct {
	Count int
	Turn  int
}

type PauseRes struct {
	Output string
}
