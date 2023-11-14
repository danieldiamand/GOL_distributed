package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"sync"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func main() {
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&GOLWorker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", "localhost:"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type GOLWorker struct {
	world    [][]byte
	turn     int
	width    int
	height   int
	isPaused bool
	worldMu  sync.Mutex
}

func (w *GOLWorker) ProgressWorld(req stubs.ProgressWorldRequest, res *stubs.ProgressWorldResponse) (err error) {
	w.world = req.World
	w.width = req.W
	w.height = req.H
	w.turn = 0
	w.isPaused = false

	println("Worker called on world", w.width, "x", w.height)
	for ; w.turn < req.Turns; w.turn++ {
		w.worldMu.Lock()
		w.world = calculateNextState(w.world, w.width, w.height)
		w.worldMu.Unlock()
	}

	res.World = w.world
	println("world progressed")

	return
}

func (w *GOLWorker) CountCells(req stubs.Empty, res *stubs.CountCellResponse) (err error) {
	w.worldMu.Lock()
	cells := 0
	for y := 0; y < w.height; y++ {
		for x := 0; x < w.width; x++ {
			if w.world[y][x] == 255 {
				cells++
			}
		}
	}
	w.worldMu.Unlock()
	res.Count = cells
	res.Turn = w.turn
	return
}

func (w *GOLWorker) Pause(req stubs.Empty, res *stubs.Empty) (err error) {
	if !w.isPaused {
		w.worldMu.Lock()
		w.isPaused = true
	} else {
		w.worldMu.Unlock()
		w.isPaused = false
	}
	return
}

//using indexing x,y where 0,0 is top left of board
func calculateNextState(world [][]byte, w, h int) [][]byte {
	newWorld := make([][]byte, h)
	for y := 0; y < h; y++ {
		newWorld[y] = make([]byte, w)
		for x := 0; x < w; x++ {
			count := liveNeighbourCount(y, x, w, h, world)
			if world[y][x] == 255 { //if cells alive:
				if count == 2 || count == 3 { //any live cell with two or three live neighbours is unaffected
					newWorld[y][x] = 255
				}
				//any live cell with fewer than two or more than three live neighbours dies
				//in go slices are initialized to zero, so we don't need to do anything
			} else { //cells dead
				if count == 3 { //any dead cell with exactly three live neighbours becomes alive
					newWorld[y][x] = 255
				}
			}
		}
	}
	return newWorld
}

func liveNeighbourCount(y, x, w, h int, world [][]byte) int8 {
	var count int8 = 0
	if world[(y+1+h)%h][(x+1+w)%w] == 255 {
		count++
	}
	if world[(y+1+h)%h][x] == 255 {
		count++
	}
	if world[(y+1+h)%h][(x-1+w)%w] == 255 {
		count++
	}
	if world[y][(x+1+w)%w] == 255 {
		count++
	}
	if world[y][(x-1+w)%w] == 255 {
		count++
	}
	if world[(y-1+h)%h][(x+1+w)%w] == 255 {
		count++
	}
	if world[(y-1+h)%h][x] == 255 {
		count++
	}
	if world[(y-1+h)%h][(x-1+w)%w] == 255 {
		count++
	}
	return count
}
