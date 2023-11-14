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
	err := rpc.Register(&GOLWorkerCommand{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", "localhost:"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type GOLWorkerCommand struct {
	world    [][]byte
	turn     int
	width    int
	height   int
	isPaused bool
	worldMu  sync.Mutex
}

func (s *GOLWorkerCommand) WorkerProgressWorld(req stubs.Request, res *stubs.Response) (err error) {
	s.world = req.World
	s.width = req.W
	s.height = req.H
	s.turn = 0

	println("Worker called on world", s.width, "x", s.height)
	for ; s.turn < req.Turns; s.turn++ {
		s.worldMu.Lock()
		s.world = calculateNextState(s.world, s.width, s.height)
		s.worldMu.Unlock()
	}

	res.World = s.world
	println("world progressed")

	return
}

func (s *GOLWorkerCommand) WorkerCountCells(req stubs.Empty, res *stubs.CountCellResponse) (err error) {
	s.worldMu.Lock()
	cells := 0
	for y := 0; y < s.height; y++ {
		for x := 0; x < s.width; x++ {
			if s.world[y][x] == 255 {
				cells++
			}
		}
	}
	s.worldMu.Unlock()
	res.Count = cells
	res.Turn = s.turn
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
