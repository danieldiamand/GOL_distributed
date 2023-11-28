package main

import (
	"flag"
	"math/rand"
	"net"
	"net/rpc"
	"os"
	"time"
	"uk.ac.bris.cs/gameoflife/stubs"
	"uk.ac.bris.cs/gameoflife/util"
)

func main() {
	pAddr := flag.String("address", ":8030", "Address to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Worker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Worker struct {
	world  [][]byte
	turn   int
	width  int
	height int
}

func (w *Worker) ProgressWorld(req stubs.WorkerProgressWorldReq, res *stubs.WorldRes) (err error) {
	w.world = [][]byte{}
	w.world = append(w.world, req.WorldTop)
	w.world = append(w.world, req.WorldMiddle...)
	w.world = append(w.world, req.WorldBot)

	w.width = req.Width
	w.height = req.Height

	println("Created worker starting work", len(w.world))

	newWorld := calculateNextState(w.world, w.width, w.height)

	//util.VisualiseMatrix(newWorld, w.width, w.height)
	res.World = newWorld
	res.Turn = w.turn

	return
}

func (w *Worker) Quit(req stubs.Empty, res *stubs.Empty) (err error) {
	println("Worker quit.")
	return
}

func (w *Worker) Kill(req stubs.Empty, res *stubs.Empty) (err error) {
	println("Worker killed.")
	os.Exit(0)
	return
}

//using indexing x,y where 0,0 is top left of board
func calculateNextState(world [][]byte, width, height int) [][]byte {

	newWorld := make([][]byte, height)
	for y := 1; y < height+1; y++ {
		newWorld[y-1] = make([]byte, width)
		for x := 0; x < width; x++ {
			count := liveNeighbourCount(y, x, width, world)
			if world[y][x] == 255 { //if cells alive:
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
	return newWorld
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
