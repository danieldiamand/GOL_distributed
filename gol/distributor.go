package gol

import (
	"fmt"
	"net/rpc"
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
func distributor(p Params, c distributorChannels) {
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

	request := stubs.Request{World: world, W: p.ImageWidth, H: p.ImageHeight, Turns: p.Turns}
	response := new(stubs.Response)
	err := client.Call(stubs.ProgressWorldHandler, request, response)
	util.HandleError(err)

	world = response.World

	//Send final world to io
	sendWorldToPGM(world, p.Turns, p, c)
	c.events <- FinalTurnComplete{p.Turns, calculateAliveCells(world, p)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

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
