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
	pAddr := flag.String("address", "localhost:8032", "Address to listen on")
	flag.Parse()
	rand.Seed(time.Now().UnixNano())
	err := rpc.Register(&Broker{})
	util.HandleError(err)
	listener, _ := net.Listen("tcp", *pAddr)
	defer listener.Close()
	rpc.Accept(listener)
}

type Broker struct {
	workers        []*rpc.Client
	workersAdr     []string
	workerSections []int
	workerCount    int

	savedWorld [][]byte
	savedTurn  int
	finalTurn  int

	width    int
	height   int
	isPaused bool
	isQuit   bool
	worldMu  sync.Mutex
}

func (b *Broker) ProgressWorld(req stubs.BrokerProgressWorldReq, res *stubs.WorldRes) (err error) {
	b.savedWorld = req.World
	b.savedTurn = 0
	b.finalTurn = req.Turns
	b.width = req.Width
	b.height = req.Height

	//Try and connect to each worker
	b.workersAdr = req.WorkersAdr
	b.workerCount = 0
	for _, workerAdr := range b.workersAdr {
		worker, err := rpc.Dial("tcp", workerAdr)
		if err != nil {
			/*
				TODO: handle this error properly, should ask user if they want to
				work with one less user or try reconnecting

				TOOD: rather than having distrubotr having an address have
				different types of response!!!
			*/
			println("Error connecting to worker", err.Error())
		}
		b.workers = append(b.workers, worker)
		b.workerCount++
	}

	//Divide board up into sections to give to each worker,
	//worker i starts at workerSections[i] and finishes at workerSections[i+1]
	sectionLength := b.height / b.workerCount
	remainingLength := b.height % b.workerCount
	b.workerSections = make([]int, b.workerCount+1)
	b.workerSections[0] = 0
	for i := 1; i < b.workerCount+1; i++ {
		b.workerSections[i] = b.workerSections[i-1] + sectionLength
		if i <= remainingLength {
			b.workerSections[i]++
		}
	}

	workerDones := make([]*rpc.Call, b.workerCount)

	//Call Init on each worker
	for i := 0; i < b.workerCount; i++ {
		workerInitReq := stubs.WorkerInitReq{
			World:  b.savedWorld[b.workerSections[i]:b.workerSections[i+1]],
			Width:  b.width,
			Height: b.workerSections[i+1] - b.workerSections[i],
		}
		workerDones[i] = b.workers[i].Go(stubs.WorkerInit, workerInitReq, &stubs.None{}, nil)
	}
	//ensure each init has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker init err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
	}

	//Call Start on each worker
	println(b.workersAdr[b.workerCount-1])
	workerStartReq := stubs.WorkerStartReq{AboveAdr: b.workersAdr[b.workerCount-1]} //the top worker connects to the bottom
	workerDones[0] = b.workers[0].Go(stubs.WorkerStart, workerStartReq, &stubs.None{}, nil)
	for i := 1; i < b.workerCount; i++ {
		println(b.workersAdr[i-1])
		workerStartReq = stubs.WorkerStartReq{AboveAdr: b.workersAdr[i-1]} //all other workers connect to the one above
		workerDones[i] = b.workers[i].Go(stubs.WorkerStart, workerStartReq, &stubs.None{}, nil)
	}
	//ensure each start has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker start err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
	}

	//MAIN LOOP:
	turn := 0
	workerTurnRes := make([]stubs.Turn, b.workerCount)
	for turn < b.finalTurn {
		//Call progressHelper on each worker
		for i := 0; i < b.workerCount; i++ {
			workerDones[i] = b.workers[i].Go(stubs.WorkerProgress, stubs.None{}, &workerTurnRes[i], nil)
		}
		//ensure each start has completed
		for i := 0; i < b.workerCount; i++ {
			if workerDones[i].Error != nil {
				println("Worker progressHelper err", workerDones[i].Error.Error())
				os.Exit(1)
			}
			<-workerDones[i].Done
		}

		turn = workerTurnRes[0].Turn
	}

	//Collect final world from workers
	workerFetchRes := make([]stubs.WorldRes, b.workerCount)
	for i := 0; i < b.workerCount; i++ {
		workerDones[i] = b.workers[i].Go(stubs.WorkerFetch, stubs.None{}, &workerFetchRes[i], nil)
	}

	var finalWorld [][]byte
	//ensure each fetch has completed
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("Worker fetch err", workerDones[i].Error.Error())
			os.Exit(1)
		}
		<-workerDones[i].Done
		finalWorld = append(finalWorld, workerFetchRes[i].World...)
	}
	finalTurn := workerFetchRes[0].Turn

	res.World = finalWorld
	res.Turn = finalTurn

	println("Broker finished progressing world")

	return
}

func (b *Broker) Count(req stubs.None, res *stubs.CountCellRes) (err error) {
	workerCountRes := make([]stubs.CountCellRes, b.workerCount)
	workerDones := make([]*rpc.Call, b.workerCount)
	for i := 0; i < b.workerCount; i++ {
		workerDones[i] = b.workers[i].Go(stubs.WorkerCount, stubs.None{}, &workerCountRes[i], nil)
	}

	count := 0
	for i := 0; i < b.workerCount; i++ {
		if workerDones[i].Error != nil {
			println("count err", workerDones[i].Error.Error())
		}
		<-workerDones[i].Done
		count += workerCountRes[i].Count
		println("worker", i, "on turn", workerCountRes[i].Turn)
	}

	res.Count = count
	res.Turn = workerCountRes[0].Turn
	return
}

//func (b *Broker) Pause(req stubs.Empty, res *stubs.PauseRes) (err error) {
//	if !b.isPaused {
//		println("Pausing on turn", b.turn)
//		b.worldMu.Lock()
//		b.isPaused = true
//		res.Output = fmt.Sprintf("Pausing on turn %d", b.turn)
//	} else {
//		println("Unpausing")
//		b.isPaused = false
//		b.worldMu.Unlock()
//		res.Output = "Continuing"
//	}
//	return
//}

//func (b *Broker) FetchWorld(req stubs.Empty, res *stubs.WorldRes) (err error) {
//	b.worldMu.Lock()
//	res.World = b.world
//	res.Turn = b.turn
//	b.worldMu.Unlock()
//	return
//}

//func (b *Broker) Quit(req stubs.Empty, res *stubs.Empty) (err error) {
//	//Send quit command to all workers
//	for _, worker := range b.workers {
//		err := worker.Call(stubs.WorkerQuit, stubs.Empty{}, &stubs.Empty{})
//		if err != nil {
//			println("worker quiting err", err.Error())
//		}
//	}
//	println("Quit all workers.")
//
//	//Quit this command
//	println("Broker quit.")
//	b.isQuit = true
//	return
//}
//
//func (b *Broker) Kill(req stubs.Empty, res *stubs.Empty) (err error) {
//	//Send kill command to all workers
//	for _, worker := range b.workers {
//		_ = worker.Go(stubs.WorkerKill, stubs.Empty{}, &stubs.Empty{}, nil)
//	}
//	println("Killed all workers.")
//
//	println("Broker killed.")
//	os.Exit(0)
//	return
//}
