package stubs

var ProgressWorldHandler = "GOLWorker.ProgressWorld"
var CountCellHandler = "GOLWorker.CountCells"
var PauseHandler = "GOLWorker.Pause"

type Empty struct {
}

type ProgressWorldRequest struct {
	World [][]byte
	W     int
	H     int
	Turns int
}

type ProgressWorldResponse struct {
	World [][]byte
}

type CountCellResponse struct {
	Count int
	Turn  int
}

type PauseResponse struct {
	Output string
}
