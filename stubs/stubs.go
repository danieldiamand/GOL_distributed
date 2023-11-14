package stubs

var ProgressWorldHandler = "GOLWorkerCommand.WorkerProgressWorld"
var CountCellHandler = "GOLWorkerCommand.WorkerCountCells"

type Response struct {
	World [][]byte
}

type Request struct {
	World [][]byte
	W     int
	H     int
	Turns int
}

type Empty struct {
}

type CountCellResponse struct {
	Count int
	Turn  int
}
