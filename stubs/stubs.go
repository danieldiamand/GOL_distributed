package stubs

var ProgressWorldHandler = "GOLWorker.ProgressWorld"
var CountCellHandler = "GOLWorker.CountCells"
var PauseHandler = "GOLWorker.Pause"
var FetchWorldHandler = "GOLWorker.FetchWorld"
var QuitHandler = "GOLWorker.Quit"
var KillHandler = "GOLWorker.Kill"

type Empty struct {
}

type ProgressWorldRequest struct {
	World [][]byte
	W     int
	H     int
	Turns int
}

type WorldResponse struct {
	World [][]byte
	Turn  int
}

type CountCellResponse struct {
	Count int
	Turn  int
}

type PauseResponse struct {
	Output string
}
