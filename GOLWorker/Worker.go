package main

import (
	"flag"
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
	util.HandleError(err)
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Worker struct {
	topHalo     []byte
	world       [][]byte
	botHalo     chan []byte
	worldMu     sync.Mutex
	worldChan   chan [][]byte
	turn        int
	width       int
	height      int
	workerAbove *rpc.Client
}

// Init : Called by Broker to first place data inside a worker
func (w *Worker) Init(req stubs.WorkerInitReq, res *stubs.None) (err error) {
	println("Worker created.")
	w.world = req.World
	w.turn = 0
	w.width = req.Width
	w.height = req.Height
	w.worldChan = make(chan [][]byte, 1)
	w.botHalo = make(chan []byte, 1)
	//println("matrix on init. w:", w.width, "h:", w.height)
	//util.VisualiseMatrix(w.world, w.width, w.height)
	return
}

// Start : Called by Broker once to start communication between workers
func (w *Worker) Start(req stubs.WorkerStartReq, res *stubs.None) (err error) {
	//Connect with worker above
	w.workerAbove, err = rpc.Dial("tcp", req.AboveAdr)
	if err != nil {
		println("Error connecting to worker above", err.Error())
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
	//println("matrix on turn", w.turn)
	//util.VisualiseMatrix(w.world, w.width, w.height)
	w.worldMu.Unlock()

	//Send receive neighbours halo/send world to calculate
	progressHelper(w)
	res.Turn = w.turn
	return
}

// progressHelper : helper command
func progressHelper(w *Worker) {
	//Share+Get halo region w neighbour above
	topHaloRes := stubs.WorkerHaloReqRes{}
	err := w.workerAbove.Call(stubs.WorkerHalo, stubs.WorkerHaloReqRes{Halo: w.world[0]}, &topHaloRes)
	if err != nil {
		println("Error doing Halo exchange", err.Error())
	}

	//Ensure we have received halo region from neighbour below
	topHalo := topHaloRes.Halo
	botHalo := <-w.botHalo

	//Start calculating first turn
	go calculateNextState(w.world, topHalo, botHalo, w.worldChan, w.width, w.height)
}

// Halo : Called by below neighbour Worker to exchange halo regions.
// each worker should call this on their neighbour above and have it called on them by there neighbour below
func (w *Worker) Halo(req stubs.WorkerHaloReqRes, res *stubs.WorkerHaloReqRes) (err error) {
	//Receive top from Worker below
	w.botHalo <- req.Halo
	//Send bottom of this worker to Worker below
	res.Halo = w.world[w.height-1]
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
	//TODO: is this a race condition, is the world actively changing???
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
func calculateNextState(world [][]byte, topPad, botPad []byte, worldChan chan<- [][]byte, width, height int) {
	var oldWorld [][]byte
	oldWorld = append(oldWorld, topPad)
	oldWorld = append(oldWorld, world...)
	oldWorld = append(oldWorld, botPad)
	newWorld := make([][]byte, height)
	for y := 1; y < height+1; y++ {
		newWorld[y-1] = make([]byte, width)
		for x := 0; x < width; x++ {
			count := liveNeighbourCount(y, x, width, oldWorld)
			if oldWorld[y][x] == 255 { //if cells alive:-
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
