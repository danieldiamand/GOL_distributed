package gol

import (
	"fmt"
	"net/rpc"
	"time"
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
	println("here")
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)
	println("HERE")

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

	server := "127.0.0.1:8030"
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	worldRequest := stubs.ProgressWorldRequest{World: world, W: p.ImageWidth, H: p.ImageHeight, Turns: p.Turns}
	worldResponse := new(stubs.WorldResponse)
	doneProgressing := client.Go(stubs.ProgressWorldHandler, worldRequest, &worldResponse, nil)

	if doneProgressing.Error != nil {
		println("Error:", doneProgressing.Error)
	}

	timer := time.NewTimer(2 * time.Second)
	done := false
	killed := false
	for {
		select {
		case <-doneProgressing.Done:
			if doneProgressing.Error != nil {
				println("progressing world err:", doneProgressing.Error.Error())
			}
			done = true
			break
		case <-timer.C:
			timer.Reset(2 * time.Second)
			countResponse := new(stubs.CountCellResponse)
			err := client.Call(stubs.CountCellHandler, stubs.Empty{}, &countResponse)
			if err != nil {
				println("err!:", err.Error())
			}
			if countResponse.Count != -1 {
				c.events <- AliveCellsCount{countResponse.Turn, countResponse.Count}
			}
			break
		case key := <-keyPresses:
			switch key {
			case 's':
				worldResponse := new(stubs.WorldResponse)
				err := client.Call(stubs.FetchWorldHandler, stubs.Empty{}, &worldResponse)
				if err != nil {
					println("fetch world err", err)
				}
				sendWorldToPGM(worldResponse.World, worldResponse.Turn, p, c)
				break
			case 'p':
				pauseResponse := new(stubs.PauseResponse)
				err := client.Call(stubs.PauseHandler, stubs.Empty{}, &pauseResponse)
				if err != nil {
					println("pausing err", err)
				}
				println(pauseResponse.Output)
				break
			case 'q':
				err := client.Call(stubs.QuitHandler, stubs.Empty{}, &stubs.Empty{})
				if err != nil {
					println("quiting err", err.Error())
				}
				println("Quiting...")
				break
			case 'k':
				err := client.Call(stubs.QuitHandler, stubs.Empty{}, &stubs.Empty{})
				if err != nil {
					println("killing err", err.Error())
				}
				println("Killing...")
				killed = true
				break
			}
			break
		}
		if done {
			break
		}
	}

	if killed {
		_ = client.Go(stubs.KillHandler, stubs.Empty{}, &stubs.Empty{}, nil)
	}

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

//Returns the number of alive neighbours of a given cell
func countNeighbours(world [][]byte, width, y, x int) int8 {
	var count int8 = 0
	if world[y+1][(x+1+width)%width] == 255 {
		count++
	}
	if world[y+1][x] == 255 {
		count++
	}
	if world[y+1][(x-1+width)%width] == 255 {
		count++
	}
	if world[y][(x+1+width)%width] == 255 {
		count++
	}
	if world[y][(x-1+width)%width] == 255 {
		count++
	}
	if world[y-1][(x+1+width)%width] == 255 {
		count++
	}
	if world[y+1][x] == 255 {
		count++
	}
	if world[y-1][(x-1+width)%width] == 255 {
		count++
	}

	return count
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

//Counts live cells in world passed to it
func countLiveCells(world [][]byte, width, height int) int {
	count := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if world[y][x] == 255 {
				count++
			}
		}
	}
	return count
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
