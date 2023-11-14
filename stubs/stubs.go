package stubs

var ProgressWorldHandler = "GOLWorker.ProgressWorld"
var CountCellHandler = "GOLWorker.CountCells"
var PauseHandler = "GOLWorker.Pause"

type ProgressWorldRequest struct {
	World [][]byte
	W     int
	H     int
	Turns int
}

type ProgressWorldResponse struct {
	World [][]byte
}

type Empty struct {
}

type CountCellResponse struct {
	Count int
	Turn  int
}
