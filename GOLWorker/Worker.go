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
	pAddr := flag.String("port", "8031", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Worker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", "localhost:"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Worker struct {
	world    [][]byte
	turn     int
	width    int
	height   int
	isPaused bool
	isQuit   bool
	worldMu  sync.Mutex
}

func (w *Worker) ProgressWorld(req stubs.WorkerProgressWorldReq, res *stubs.WorldRes) (err error) {
	w.world = req.World
	w.width = req.W
	w.height = req.H

	newWorld := calculateNextState(w.world, w.width, w.height)

	res.World = newWorld
	res.Turn = w.turn

	return
}

//
//func (w *Worker) CountCells(req stubs.Empty, res *stubs.CountCellResponse) (err error) {
//	if w.isPaused {
//		res.Count = -1
//		return
//	}
//	w.worldMu.Lock()
//	cells := 0
//	for y := 0; y < w.height; y++ {
//		for x := 0; x < w.width; x++ {
//			if w.world[y][x] == 255 {
//				cells++
//			}
//		}
//	}
//	w.worldMu.Unlock()
//	res.Count = cells
//	res.Turn = w.turn
//	return
//}
//
//func (w *Worker) Pause(req stubs.Empty, res *stubs.PauseResponse) (err error) {
//	if !w.isPaused {
//		println("Pausing on turn", w.turn)
//		w.worldMu.Lock()
//		w.isPaused = true
//		res.Output = fmt.Sprintf("Pausing on turn %d", w.turn)
//	} else {
//		println("Unpausing")
//		w.isPaused = false
//		w.worldMu.Unlock()
//		res.Output = "Continuing"
//	}
//	return
//}
//
//func (w *Worker) FetchWorld(req stubs.Empty, res *stubs.WorldResponse) (err error) {
//	w.worldMu.Lock()
//	res.World = w.world
//	res.Turn = w.turn
//	w.worldMu.Unlock()
//	return
//}
//
//func (w *Worker) Quit(req stubs.Empty, res *stubs.Empty) (err error) {
//	println("Worker quit.")
//	w.isQuit = true
//	return
//}
//
//func (w *Worker) Kill(req stubs.Empty, res *stubs.Empty) (err error) {
//	println("Worker killed.")
//	os.Exit(0)
//	return
//}

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
