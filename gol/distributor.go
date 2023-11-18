package gol

import (
	"fmt"
	"net/rpc"
	"strings"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

type distributorChannels struct {
	events     chan<- Event
	ioCommand  chan<- ioCommand
	ioIdle     <-chan bool
	ioFilename chan<- string
	ioOutput   chan<- uint8
	ioInput    <-chan uint8
}

type workerChannels struct {
	outTop         chan<- []byte
	outBottom      chan<- []byte
	inTop          <-chan []byte
	inBottom       <-chan []byte
	world          chan [][]byte
	aliveCells     chan []util.Cell
	aliveCellCount chan turnCount
	distribEvent   chan<- Event
}

type turnCount struct {
	turn  int
	Count int
}

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels, keyPresses <-chan rune) {
	//Activate IO to output world:
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)

	//Create 2D slice and store received world in it, also send live cells down cell flipped
	world := make([][]byte, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		world[y] = make([]byte, p.ImageWidth)
		for x := 0; x < p.ImageWidth; x++ {
			cell := <-c.ioInput
			if cell == 255 {
				c.events <- CellFlipped{0, util.Cell{x, y}}
			}
			world[y][x] = cell
		}
	}

	//Connect to broker
	broker, err := rpc.Dial("tcp", p.BrokerAddress)
	if err != nil {
		println("connecting to broker Err", err.Error())
	}
	defer func(broker *rpc.Client) {
		err := broker.Close()
		if err != nil {
			println("error closing broker", err.Error())
		}
	}(broker)

	workerAddresses := strings.Split(p.WorkerAddresses, ",")
	println(workerAddresses)

	worldRequest := stubs.BrokerProgressWorldReq{World: world, Width: p.ImageWidth, Height: p.ImageHeight, Turns: p.Turns, WorkersAdr: workerAddresses}
	worldResponse := new(stubs.WorldRes)
	doneProgressing := broker.Go(stubs.BrokerProgressWorld, worldRequest, &worldResponse, nil)

	if doneProgressing.Error != nil {
		println("Error:", doneProgressing.Error)
	}

	//timer := time.NewTimer(2 * time.Second)
	done := false
	//killed := false
	for !done {
		select {
		case <-doneProgressing.Done:
			if doneProgressing.Error != nil {
				println("progressing world err:", doneProgressing.Error.Error())
			}
			done = true
			break
			//case <-timer.C:
			//	timer.Reset(2 * time.Second)
			//	countResponse := new(stubs.CountCellRes)
			//	err := broker.Call(stubs.BrokerCountCells, stubs.Empty{}, &countResponse)
			//	if err != nil {
			//		println("err!:", err.Error())
			//	}
			//	if countResponse.Count != -1 {
			//		c.events <- AliveCellsCount{countResponse.Turn, countResponse.Count}
			//	}
			//	break
			//case key := <-keyPresses:
			//	switch key {
			//	case 's':
			//		worldResponse := new(stubs.WorldRes)
			//		err := broker.Call(stubs.BrokerFetchWorld, stubs.Empty{}, &worldResponse)
			//		if err != nil {
			//			println("fetch world err", err)
			//		}
			//		sendWorldToPGM(worldResponse.World, worldResponse.Turn, p, c)
			//		break
			//	case 'p':
			//		pauseResponse := new(stubs.PauseRes)
			//		err := broker.Call(stubs.BrokerPause, stubs.Empty{}, &pauseResponse)
			//		if err != nil {
			//			println("pausing err", err)
			//		}
			//		println(pauseResponse.Output)
			//		break
			//	case 'q':
			//		err := broker.Call(stubs.BrokerQuit, stubs.Empty{}, &stubs.Empty{})
			//		if err != nil {
			//			println("quiting err", err.Error())
			//		}
			//		println("Quiting...")
			//		break
			//	case 'k':
			//		err := broker.Call(stubs.BrokerQuit, stubs.Empty{}, &stubs.Empty{})
			//		if err != nil {
			//			println("killing err", err.Error())
			//		}
			//		println("Killing...")
			//		killed = true
			//		break

		}
	}

	//if killed {
	//	_ = broker.Go(stubs.BrokerKill, stubs.Empty{}, &stubs.Empty{}, nil)
	//}

	world = worldResponse.World
	finalTurn := worldResponse.Turn

	//Send final world to io
	sendWorldToPGM(world, finalTurn, p, c)
	c.events <- FinalTurnComplete{finalTurn, calculateAliveCells(world, p)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{finalTurn, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}

//Returns list of all alive cells in board
func calculateAliveCells(world [][]byte, p Params) []util.Cell {
	cells := make([]util.Cell, 0)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			if world[y][x] == 255 {
				cells = append(cells, util.Cell{x, y})
			}
		}
	}
	return cells
}

//Prepares io for output and sends board down it a pixel at a time
func sendWorldToPGM(world [][]byte, turn int, p Params, c distributorChannels) {
	c.ioCommand <- ioOutput
	c.ioFilename <- fmt.Sprintf("%dx%dx%d", p.ImageHeight, p.ImageWidth, turn)
	for y := 0; y < p.ImageHeight; y++ {
		for x := 0; x < p.ImageWidth; x++ {
			c.ioOutput <- world[y][x]
		}
	}
}
