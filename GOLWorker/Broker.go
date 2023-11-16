package main

import (
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
	pAddr := flag.String("port", "8030", "Port to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Broker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", "localhost:"+*pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Broker struct {
	world    [][]byte
	workers  []*rpc.Client
	turn     int
	width    int
	height   int
	isPaused bool
	isQuit   bool
	worldMu  sync.Mutex
}

func (b *Broker) ProgressWorld(progressReq stubs.BrokerProgressWorldReq, progressRes *stubs.WorldRes) (err error) {
	//Try and connect to all workers
	for _, workerAdr := range progressReq.WorkersAdr {
		worker, err := rpc.Dial("tcp", workerAdr)
		if err != nil {
			println("Error connecting to worker", err.Error())
		}
		b.workers = append(b.workers, worker)
	}

	b.world = progressReq.World
	b.width = progressReq.W
	b.height = progressReq.H
	b.turn = 0
	b.isQuit = false
	b.isPaused = false

	println("Broker progressing world: ", b.width, "x", b.height)
	for b.turn < progressReq.Turns && !b.isQuit {
		worldRequest := stubs.WorkerProgressWorldReq{World: b.world, W: b.width, H: b.height}
		worldResponse := new(stubs.WorldRes)
		err := b.workers[0].Call(stubs.WorkerProgressWorld, worldRequest, &worldResponse)

		if err != nil {
			println("worker progress world err", err.Error())
		}

		b.worldMu.Lock()
		b.world = worldResponse.World
		b.turn++
		b.worldMu.Unlock()
	}

	progressRes.World = b.world
	progressRes.Turn = b.turn
	println("Broker finished progressing world")

	return
}

func (b *Broker) CountCells(req stubs.Empty, res *stubs.CountCellRes) (err error) {
	if b.isPaused {
		res.Count = -1
		return
	}

	b.worldMu.Lock()
	cells := 0
	for y := 0; y < b.height; y++ {
		for x := 0; x < b.width; x++ {
			if b.world[y][x] == 255 {
				cells++
			}
		}
	}
	b.worldMu.Unlock()
	res.Count = cells
	res.Turn = b.turn
	return
}

func (b *Broker) Pause(req stubs.Empty, res *stubs.PauseRes) (err error) {
	if !b.isPaused {
		println("Pausing on turn", b.turn)
		b.worldMu.Lock()
		b.isPaused = true
		res.Output = fmt.Sprintf("Pausing on turn %d", b.turn)
	} else {
		println("Unpausing")
		b.isPaused = false
		b.worldMu.Unlock()
		res.Output = "Continuing"
	}
	return
}

func (b *Broker) FetchWorld(req stubs.Empty, res *stubs.WorldRes) (err error) {
	b.worldMu.Lock()
	res.World = b.world
	res.Turn = b.turn
	b.worldMu.Unlock()
	return
}

func (b *Broker) Quit(req stubs.Empty, res *stubs.Empty) (err error) {
	//Send quit command to all workers
	for _, worker := range b.workers {
		err := worker.Call(stubs.WorkerQuit, stubs.Empty{}, &stubs.Empty{})
		if err != nil {
			println("worker quiting err", err.Error())
		}
		break
	}
	//Quit this command
	println("Broker quit.")
	b.isQuit = true
	return
}

func (b *Broker) Kill(req stubs.Empty, res *stubs.Empty) (err error) {
	//Send kill command to all workers
	for _, worker := range b.workers {
		err := worker.Call(stubs.WorkerKill, stubs.Empty{}, &stubs.Empty{})
		if err != nil {
			println("Worker killing err", err.Error())
		}
		break
	}

	println("Broker killed.")
	os.Exit(0)
	return
}
