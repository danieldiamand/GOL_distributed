package stubs

var WorkerProgressWorld = "Worker.ProgressWorld"
var WorkerCountCells = "Worker.CountCells"
var WorkerPause = "Worker.Pause"
var WorkerFetchWorld = "Worker.FetchWorld"
var WorkerQuit = "Worker.Quit"
var WorkerKill = "Worker.Kill"

var BrokerProgressWorld = "Broker.ProgressWorld"

type Empty struct {
}

type BrokerProgressWorldReq struct {
	WorkersAdr []string
	World      [][]byte
	W          int
	H          int
	Turns      int
}

type WorkerProgressWorldReq struct {
	World [][]byte
	W     int
	H     int
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
