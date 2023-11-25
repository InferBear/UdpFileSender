package main

import (
	"UdpFileSender/common"
	"encoding/json"
	"log"
	"math"
	"net"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

const FileBlockSize = 1024
const Concurrency = 50

var fileLock sync.Mutex
var wg sync.WaitGroup
var retriedCount = atomic.Int32{}
var totalWrote = int64(0)

func sendRequest(fileReq common.FileRequest, server string) common.FileResponse {
	conn, err := net.Dial("udp", server)
	if err != nil {
		log.Fatalf("Error dialing UDP: %v", err)
	}
	defer conn.Close()

	req, err := json.Marshal(fileReq)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v", err)
	}

	_, err = conn.Write(req)
	if err != nil {
		log.Fatalf("Error writing to UDP connection: %v", err)
	}

	var buf [4096]byte
	n, err := conn.Read(buf[0:])
	if err != nil {
		log.Fatalf("Error reading from UDP connection: %v", err)
	}

	var fileResp common.FileResponse
	err = json.Unmarshal(buf[0:n], &fileResp)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON: %v", err)
	}

	return fileResp

}

func getFileSize(server string) int64 {
	log.Println("Requesting file size")
	fileReq := common.FileRequest{Start: math.MaxInt64, End: math.MaxInt64}
	response := make(chan common.FileResponse, 1)
	for {
		timer := time.NewTimer(5 * time.Second)
		go func() {
			data := sendRequest(fileReq, server)
			response <- data
		}()

		select {
		case <-timer.C:
			continue
		case result := <-response:
			return result.Start
		}
	}
}

func requestFileBlock(region, totalBlock int64, file *os.File, server string, windowChannel chan int64) {
	for blockId := region; blockId < totalBlock; blockId += Concurrency {
		startFile, endFile := blockId*FileBlockSize, (blockId+1)*FileBlockSize
		data := requestFileContent(startFile, endFile, file, server)
		go saveFileBlock(startFile, endFile, totalBlock, file, data)
		log.Printf("Requested file block finished: %d\n", blockId)
	}
}

func requestFileContent(start, end int64, file *os.File, server string) []byte {
	fileReq := common.FileRequest{Start: start, End: end}
	responseChan := make(chan common.FileResponse, 1)
	for {
		timer := time.NewTimer(time.Duration(time.Millisecond * 500))
		go func() {
			data := sendRequest(fileReq, server)
			responseChan <- data
		}()
		select {
		case res := <-responseChan:
			return res.Content
		case <-timer.C:
			retriedCount.Add(1)
			log.Printf("File content request timed out, retrying, total Retries(%d)", retriedCount)
			return requestFileContent(start, end, file, server)
		}
	}
}

func saveFileBlock(start, end, totalBlock int64, file *os.File, data []byte) {
	fileLock.Lock()
	defer fileLock.Unlock()
	file.Seek(start, 0)
	file.Write(data)
	totalWrote += 1
	if totalWrote == totalBlock {
		wg.Done()
	}
	log.Printf("Saving file block from %d to %d\n", start, end)
}

func saveFile(savePath string, server string) {
	fileSize := getFileSize(server)
	file, _ := os.OpenFile(savePath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	totalBlock := fileSize / FileBlockSize
	windowSize := min(totalBlock, Concurrency)
	wg.Add(1)
	windowChannel := make(chan int64, windowSize)
	for i := int64(0); i < windowSize; i++ {
		go requestFileBlock(i, totalBlock, file, server, windowChannel)
	}
	wg.Wait()

	if fileSize%FileBlockSize != 0 {
		startOffset := fileSize - fileSize%FileBlockSize
		data := requestFileContent(startOffset, fileSize, file, server)
		saveFileBlock(startOffset, fileSize, totalBlock, file, data)
	}

	log.Printf("TotalBlock: %d\n, RetriesCount: (%d) ", totalBlock, retriedCount)
}

func main() {
	saveFile("file.txt", "127.0.0.1:9991")
}

func min(a int64, b int64) int64 {
	if a > b {
		return b
	} else {
		return a
	}
}
