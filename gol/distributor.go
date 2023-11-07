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

// distributor divides the work between workers and interacts with other goroutines.
func distributor(p Params, c distributorChannels) {
	//Activate IO to output world:
	c.ioCommand <- ioInput
	c.ioFilename <- fmt.Sprintf("%dx%d", p.ImageHeight, p.ImageWidth)

	//Create 2D slice and store received world in it, also send live cells down cell flipped
	startWorld := make([][]byte, p.ImageHeight)
	for y := 0; y < p.ImageHeight; y++ {
		startWorld[y] = make([]byte, p.ImageWidth)
		for x := 0; x < p.ImageWidth; x++ {
			cell := <-c.ioInput
			if cell == 255 {
				c.events <- CellFlipped{0, util.Cell{x, y}}
			}
			startWorld[y][x] = cell
		}
	}

	server := "localhost:8030"
	client, _ := rpc.Dial("tcp", server)
	defer client.Close()

	request := stubs.Request{World: startWorld, W: p.ImageWidth, H: p.ImageHeight, Turns: p.Turns}
	response := new(stubs.Response)
	client.Call(stubs.ProgressWorldHandler, request, response)

	util.VisualiseMatrix(response.World, p.ImageWidth, p.ImageHeight)

	//sendWorldToPGM(safeWorld, turn, p, c)
	//c.events <- FinalTurnComplete{turn, calculateAliveCells(safeWorld, p)}

	// Make sure that the Io has finished any output before exiting.
	c.ioCommand <- ioCheckIdle
	<-c.ioIdle

	c.events <- StateChange{p.Turns, Quitting}

	// Close the channel to stop the SDL goroutine gracefully. Removing may cause deadlock.
	close(c.events)
}
