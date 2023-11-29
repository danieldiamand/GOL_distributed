package main

import (
	"errors"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func main() {
	pAddr := flag.String("address", "localhost:8031", "Address to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Worker{})
	if err != nil {
		println("Error registering worker:", err.Error())
		return
	}
	listener, err := net.Listen("tcp", *pAddr)
	if err != nil {
		println("Error listening on network:", err.Error())
		return
	}
	defer listener.Close()
	rpc.Accept(listener)
}

type Worker struct {
	topHalo       []byte
	world         [][]byte
	botHalo       chan []byte
	worldMu       sync.Mutex
	worldChan     chan [][]byte
	turn          int
	width         int
	height        int
	PrintProgress bool
	worldBuilt    chan bool
	workerAbove   *rpc.Client
}

// Init : Called by Broker to first place data inside a worker
func (w *Worker) Init(req stubs.WorkerInitReq, res *stubs.None) (err error) {
	println("Worker created.")
	w.world = req.World
	w.turn = 0
	w.width = req.Width
	w.height = req.Height
	w.worldChan = make(chan [][]byte, 1)
	w.worldBuilt = make(chan bool, 1)
	w.topHalo = []byte{}
	w.botHalo = make(chan []byte, 1)
	w.PrintProgress = req.PrintProgress
	if w.PrintProgress {
		println("On Turn", w.turn)
		util.VisualiseMatrix(w.world, w.width, w.height)
	}
	return
}

// Start : Called by Broker once to start communication between workers
func (w *Worker) Start(req stubs.WorkerStartReq, res *stubs.None) (err error) {
	//Connect with worker above
	w.worldBuilt <- true
	w.workerAbove, err = rpc.Dial("tcp", req.AboveAdr)
	if err != nil {
		println("Error in Worker connecting to Worker: ", err.Error())
		return errors.New(fmt.Sprint("Error in Worker connecting to Worker: ", err.Error()))
	}
	//Do first communication with neighbouring workers
	progressHelper(w)
	return
}

// Progress : Called by Broker to progressHelper the worker one turn
func (w *Worker) Progress(req stubs.None, res *stubs.Turn) (err error) {
	//Get world when done calculating
	w.worldMu.Lock()
	w.world = <-w.worldChan
	w.turn++
	w.worldMu.Unlock()
	w.worldBuilt <- true

	//Send receive neighbours halo/send world to calculate
	err = progressHelper(w)
	if err != nil {
		return err
	}
	res.Turn = w.turn
	return
}

// progressHelper : helper command
func progressHelper(w *Worker) (err error) {
	//Share+Get halo region w neighbour above
	topHaloRes := stubs.WorkerHaloReqRes{}
	err = w.workerAbove.Call(stubs.WorkerHalo, stubs.WorkerHaloReqRes{Halo: w.world[0]}, &topHaloRes)
	if err != nil {
		println("Error doing Halo exchange", err.Error())
		return errors.New(fmt.Sprint("Error in Worker calling Halo on Worker: ", err.Error()))
	}

	//Ensure we have received halo region from neighbour below
	topHalo := topHaloRes.Halo
	botHalo := <-w.botHalo

	//Start calculating first turn
	go calculateNextState(w.world, topHalo, botHalo, w.worldChan, w.width, w.height, w.turn, w.PrintProgress)
	return
}

// Halo : Called by below neighbour Worker to exchange halo regions.
// each worker should call this on their neighbour above and have it called on them by there neighbour below
func (w *Worker) Halo(req stubs.WorkerHaloReqRes, res *stubs.WorkerHaloReqRes) (err error) {
	<-w.worldBuilt
	//Receive top from Worker below
	w.botHalo <- req.Halo
	//Send bottom of this worker to Worker below
	w.worldMu.Lock()
	res.Halo = make([]byte, w.width)
	copy(res.Halo, w.world[w.height-1])
	w.worldMu.Unlock()
	return
}

// Count : Called by Broker to count alive cells in worker
func (w *Worker) Count(req stubs.None, res *stubs.CountCellRes) (err error) {
	w.worldMu.Lock()
	cells := 0
	for y := 0; y < w.height; y++ {
		for x := 0; x < w.width; x++ {
			if w.world[y][x] == 255 {
				cells++
			}
		}
	}
	res.Count = cells
	res.Turn = w.turn
	w.worldMu.Unlock()
	return
}

// Fetch : Called by Broker to get the world stored in worker
func (w *Worker) Fetch(req stubs.None, res *stubs.WorldRes) (err error) {
	res.World = w.world
	res.Turn = w.turn
	return
}

func (w *Worker) Quit(req stubs.None, res *stubs.None) (err error) {
	println("Worker quit.")
	w.turn = -1
	return
}

func (w *Worker) Kill(req stubs.None, res *stubs.None) (err error) {
	println("Worker killed.")
	os.Exit(0)
	return
}

//using indexing x,y where 0,0 is top left of board
func calculateNextState(world [][]byte, topPad, botPad []byte, worldChan chan<- [][]byte, width, height, turn int, printProgress bool) {
	if printProgress {
		println("On Turn", turn)
		var padBox [][]byte
		padBox = append(padBox, topPad)
		println("Top padding:")
		util.PrintMatrix(padBox, width, 1)
		println("Stored world:")
		util.PrintMatrix(world, width, height)
		var padBox2 [][]byte
		padBox2 = append(padBox2, botPad)
		println("Bottom Padding:")
		util.PrintMatrix(padBox2, width, 1)

	}

	var oldWorld [][]byte
	oldWorld = append(oldWorld, topPad)
	oldWorld = append(oldWorld, world...)
	oldWorld = append(oldWorld, botPad)
	newWorld := make([][]byte, height)
	for y := 1; y < height+1; y++ {
		newWorld[y-1] = make([]byte, width)
		for x := 0; x < width; x++ {
			count := liveNeighbourCount(y, x, width, oldWorld)
			if oldWorld[y][x] == 255 { //if cells alive:
				if count == 2 || count == 3 { //any live cell with two or three live neighbours is unaffected
					newWorld[y-1][x] = 255
				}
				//any live cell with fewer than two or more than three live neighbours dies
				//in go slices are initialized to zero, so we don't need to do anything
			} else { //cells dead
				if count == 3 { //any dead cell with exactly three live neighbours becomes alive
					newWorld[y-1][x] = 255
				}
			}
		}
	}
	worldChan <- newWorld
}

func liveNeighbourCount(y, x, w int, world [][]byte) int8 {
	var count int8 = 0
	if world[y+1][(x+1+w)%w] == 255 {
		count++
	}
	if world[y+1][x] == 255 {
		count++
	}
	if world[y+1][(x-1+w)%w] == 255 {
		count++
	}
	if world[y][(x+1+w)%w] == 255 {
		count++
	}
	if world[y][(x-1+w)%w] == 255 {
		count++
	}
	if world[y-1][(x+1+w)%w] == 255 {
		count++
	}
	if world[y-1][x] == 255 {
		count++
	}
	if world[y-1][(x-1+w)%w] == 255 {
		count++
	}
	return count
}
