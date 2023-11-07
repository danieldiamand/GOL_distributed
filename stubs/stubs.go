package stubs

var ProgressWorldHandler = "GOLWorkerCommand.WorkerProgressWorld"

type Response struct {
	World [][]byte
}

type Request struct {
	World [][]byte
	W     int
	H     int
	Turns int
}
