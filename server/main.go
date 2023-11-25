package main

import (
	"UdpFileSender/common"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"os"
)

func main() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 监听UDP端口
	addr, err := net.ResolveUDPAddr("udp", ":9991")
	if err != nil {
		log.Fatalf("Error resolving UDP address: %v", err)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Fatalf("Error listening on UDP port: %v", err)
	}

	for {
		var buf = make([]byte, 1024)
		// 读取客户端请求
		n, addr, err := conn.ReadFromUDP(buf[0:])
		go handleClient(n, addr, buf, err, conn)
	}
}

func handleClient(n int, addr *net.UDPAddr, buf []byte, err error, conn *net.UDPConn) {
	//	var buf [1024]byte

	// 读取客户端请求
	//	n, addr, err := conn.ReadFromUDP(buf[0:])
	if err != nil {
		log.Printf("Error reading from UDP connection: %v", err)
		return
	}
	var fileReq common.FileRequest
	err = json.Unmarshal(buf[0:n], &fileReq)
	if err != nil {
		log.Printf("Error unmarshaling JSON request: %v", err)
		return
	}

	// 读取文件内容
	file, err := os.Open("bible.txt")
	if err != nil {
		log.Printf("Error opening file: %v", err)
		return
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("Error getting file info: %v", err)
		return
	}
	var fileResp common.FileResponse
	var md5hash string
	if fileReq.Start == math.MaxInt64 {
		fileResp = common.FileResponse{
			Content: nil,
			MD5Hash: "fileSize",
			Start:   fileInfo.Size(),
			End:     fileInfo.Size(),
		}
	} else {
		file.Seek(int64(fileReq.Start), 0)
		content := make([]byte, fileReq.End-fileReq.Start)
		_, err := file.Read(content)
		if err != nil {
			log.Printf("Error reading file content from(%d) - to(%d): %v", fileReq.Start, fileReq.End, err)
			return
		}
		// 计算MD5
		hasher := md5.New()
		io.WriteString(hasher, string(content))
		md5hash = fmt.Sprintf("%x", hasher.Sum(nil))
		// 发送响应
		fileResp = common.FileResponse{
			Content: content,
			MD5Hash: md5hash,
			Start:   fileReq.Start,
			End:     fileReq.End,
		}
	}
	resp, err := json.Marshal(fileResp)
	if err != nil {
		log.Printf("Error marshaling JSON response: %v", err)
		return
	}
	_, err = conn.WriteToUDP(resp, addr)
	if err != nil {
		log.Printf("Error writing response to UDP connection: %v", err)
		return
	}
	log.Printf("Sent response to %s:%d, MD5: %s, Start: %d, End: %d", addr.IP, addr.Port, md5hash, fileReq.Start, fileReq.End)
}
