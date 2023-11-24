import csv
import subprocess
import time


BROKER_ADDRESS = "localhost:8032"
WORKER_ADDRESSES = ["localhost:8030", "localhost:8031"]


def getTime(threads):
    workerAddresses = ""
    for i in range(threads):
        workerAddresses += WORKER_ADDRESSES[i]
        workerAddresses += ","
        if i == threads:
            break
    workerAddresses=workerAddresses[:-1]

    start_time = time.time()
    subprocess.call(["go", "run", "main.go", "-t",str(threads), "-brokerAddress", BROKER_ADDRESS, "-workerAddresses", workerAddresses, "-turns","2000"])
    return time.time() - start_time

with open('output.csv', 'w', newline='') as file:
    writer = csv.writer(file)
    for threads in range(1, 3):
        row = [threads]
        for _ in range(10):
            row.append(getTime(threads))
        writer.writerow(row)
            
