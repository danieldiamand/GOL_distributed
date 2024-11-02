# Distributed Conway's Game of Life

This project is a distributed implementation of Conway's Game of Life, utilizing a halo exchange approach to optimize data communication between worker nodes. Each worker handles a section of the grid, and a central broker coordinates the simulation. Completed as part of coursework at the University of Bristol.

## Overview

The system consists of:
- **Broker**: Manages communication between workers.
- **Workers**: Each processes a portion of the grid, exchanging border data (halo) with neighboring workers.
- **Main Controller**: Runs the simulation using distributed workers.

## Setup and Execution

### 1. Start the Broker

Start the broker to manage worker communication:
```bash
./go run ./GOLWorker/Broker.go -address <broker_ip:port>
```

### 2. Start the Workers

Start each worker:
```bash
./go run ./GOLWorker/Worker.go -address <worker_ip:port>
```

### 3. Run the Main Program

To initiate the Game of Life simulation, run main.go with the broker address and a comma-separated list of worker addresses:

```bash
./go run main.go -brokerAddress <broker_ip:port> -workerAddresses <worker1_ip:port>,<worker2_ip:port>,...
```

## Usage
Run the program with the following flags:

- `-w <width>`: Set the width of the board.
- `-h <height>`: Set the height of the board.
- `-t <threads>`: Specify the number of workers to use.
- `-turns <turns>`: Specify the number of turns to process.
- `-printProgress <terminal output of board progress>`: Outputs the board progress to terminal.
<em>
Note:
-The program requires a matching PGM image file in `./images` for the specified width and height. If no image is found, it will not start.
-The `-t` flag must match the number of worker addresses passed in
-The `-printProgress` flag only works well on small boards.
</em>


### Example of running locally

Navigate to route directory of the project and run:
<em> Usage of `n` to be replaced with relevent number </em>

Terminal 0:
```bash
./go run ./GOLWorker/Broker.go -address :8030
```

Terminal 1-n:
```bash
./go run ./GOLWorker/Worker.go -address :803n
```

Terminal n+1:
```bash
./go run main.go -w 512 -h 512 -t n-1 -brokerAddress :8030 -workerAddresses :8031,:8032,...,:803n
```


