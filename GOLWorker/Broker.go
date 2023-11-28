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
	pAddr := flag.String("address", ":8030", "Address to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Broker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Broker struct {
	world          [][]byte
	workers        []*rpc.Client
	distributor    *rpc.Client
	workerSections []int
	turn           int
	width          int
	height         int
	isPaused       bool
	isQuit         bool
	worldMu        sync.Mutex
}

func (b *Broker) ProgressWorld(progressReq stubs.BrokerProgressWorldReq, progressRes *stubs.WorldRes) (err error) {
	b.distributor, err = rpc.Dial("tcp", "137.222.229.8:8030")
	if err != nil {
		println("error connecting to dist", err.Error())
	}

	//Try and connect to all workers
	for _, workerAdr := range progressReq.WorkersAdr {
		worker, err := rpc.Dial("tcp", workerAdr)
		if err != nil {
			println("Error connecting to worker", err.Error())
		}
		b.workers = append(b.workers, worker)
	}

	b.world = progressReq.World
	b.width = progressReq.Width
	b.height = progressReq.Height
	b.turn = 0
	b.isQuit = false
	b.isPaused = false

	//workerSections determines how the board is divided up into sections
	workerCount := len(b.workers)
	sectionLength := b.height / workerCount
	remainingLength := b.height % workerCount
	b.workerSections = make([]int, workerCount+1)
	b.workerSections[0] = 0
	for i := 1; i < workerCount+1; i++ {
		b.workerSections[i] = b.workerSections[i-1] + sectionLength
		if i <= remainingLength {
			b.workerSections[i]++
		}
	}

	//initalize worker response and done lists
	workerDones := make([]*rpc.Call, workerCount)
	workerResponses := make([]stubs.WorldRes, workerCount)

	//once there all done collect them up and create new world
	println("Broker progressing world: ", b.width, "x", b.height)
	for b.turn < progressReq.Turns && !b.isQuit {
		if workerCount == 1 {
			sectionStart := b.workerSections[0]
			sectionEnd := b.workerSections[1]
			worldRequest := stubs.WorkerProgressWorldReq{
				WorldTop:    b.world[b.height-1],
				WorldMiddle: b.world[sectionStart:sectionEnd],
				WorldBot:    b.world[0],
				Width:       b.width,
				Height:      sectionEnd - sectionStart,
			}
			workerDones[0] = b.workers[0].Go(stubs.WorkerProgressWorld, worldRequest, &workerResponses[0], nil)
		} else {
			//First worker
			sectionStart := b.workerSections[0]
			sectionEnd := b.workerSections[1]
			worldRequest := stubs.WorkerProgressWorldReq{
				WorldTop:    b.world[b.height-1],
				WorldMiddle: b.world[sectionStart:sectionEnd],
				WorldBot:    b.world[sectionEnd],
				Width:       b.width,
				Height:      sectionEnd - sectionStart,
			}
			workerDones[0] = b.workers[0].Go(stubs.WorkerProgressWorld, worldRequest, &workerResponses[0], nil)

			//Middle workers
			for i := 1; i < workerCount-1; i++ {
				sectionStart = b.workerSections[i]
				sectionEnd = b.workerSections[i+1]
				worldRequest = stubs.WorkerProgressWorldReq{
					WorldTop:    b.world[sectionStart-1],
					WorldMiddle: b.world[sectionStart:sectionEnd],
					WorldBot:    b.world[sectionEnd],
					Width:       b.width,
					Height:      sectionEnd - sectionStart,
				}
				workerDones[i] = b.workers[i].Go(stubs.WorkerProgressWorld, worldRequest, &workerResponses[i], nil)
			}

			//Final worker
			sectionStart = b.workerSections[workerCount-1]
			sectionEnd = b.workerSections[workerCount]
			worldRequest = stubs.WorkerProgressWorldReq{
				WorldTop:    b.world[sectionStart-1],
				WorldMiddle: b.world[sectionStart:sectionEnd],
				WorldBot:    b.world[0], //TODO: potentailly wrong
				Width:       b.width,
				Height:      sectionEnd - sectionStart,
			} //TODO: edit worker
			workerDones[workerCount-1] = b.workers[workerCount-1].Go(stubs.WorkerProgressWorld, worldRequest, &workerResponses[workerCount-1], nil)
		}

		//Collect work from each worker
		var newWorld [][]byte
		for i := 0; i < workerCount; i++ {
			<-workerDones[i].Done
			newWorld = append(newWorld, workerResponses[i].World...)
		}

		//util.VisualiseMatrix(newWorld, b.width, b.height)

		b.worldMu.Lock()
		b.world = newWorld
		b.turn++
		b.distributor.Go(stubs.DistRecieve, stubs.WorldRes{World: b.world, Turn: b.turn}, &stubs.Empty{}, nil)
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
	}
	println("Quit all workers.")

	//Quit this command
	println("Broker quit.")
	b.isQuit = true
	return
}

func (b *Broker) Kill(req stubs.Empty, res *stubs.Empty) (err error) {
	//Send kill command to all workers
	for _, worker := range b.workers {
		_ = worker.Go(stubs.WorkerKill, stubs.Empty{}, &stubs.Empty{}, nil)
	}
	println("Killed all workers.")

	println("Broker killed.")
	os.Exit(0)
	return
}
