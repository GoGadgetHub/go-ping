package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

type ICMP struct {
	Type           uint8
	Code           uint8
	Checksum       uint16
	Identifier     uint16
	SequenceNumber uint16
}

/*
- -t timeout (ms)
- -l buffered size
- -c count
*/
var (
	timeout int64
	size    int
	count   int
	typ     uint8 = 8
	code    uint8 = 0
)

func getCommandArgs() {
	flag.Int64Var(&timeout, "t", 30000, "等待时间")
	flag.IntVar(&size, "l", 54, "缓存区大小")
	flag.IntVar(&count, "c", 0, "请求次数")
	flag.Parse()
}

func getDesIp() string {
	return os.Args[len(os.Args)-1]
}

func data2bytes(icmp any, size int) (buffer bytes.Buffer, data []byte) {
	err := binary.Write(&buffer, binary.BigEndian, icmp)
	if err != nil {
		log.Fatal(err.Error())
		return
	}
	data = make([]byte, size)
	buffer.Write(data)
	data = buffer.Bytes()
	return buffer, data
}

func connectICMP(network string, times time.Duration) net.Conn {
	conn, err := net.DialTimeout(network, getDesIp(), times)
	if err != nil {
		log.Fatal("超时：", err)
		return nil
	}
	return conn
}

func main() {
	getCommandArgs()
	fmt.Printf("timeout: %d; size: %d; count: %d;\n", timeout, size, count)

	var times = time.Duration(timeout) * time.Millisecond
	conn := connectICMP("ip:icmp", times)
	defer conn.Close()

	remoteAddr := conn.RemoteAddr()
	fmt.Println(remoteAddr)

	fmt.Printf("PING %s (%s): %d data bytes\n", getDesIp(), remoteAddr, size)
	for i := 0; i < count; i++ {
		icmp := ICMP{
			typ,
			code,
			0,
			uint16(i),
			uint16(i),
		}
		_, data := data2bytes(icmp, size)
		checksum := getChecksum(data)
		data[2] = byte(checksum >> 8)
		data[3] = byte(checksum)
		err := conn.SetDeadline(time.Now().Add(time.Duration(timeout)))
		if err != nil {
			break
		}
		_, err = conn.Write(data)
		if err != nil {
			log.Println(err)
			break
		}
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			log.Println(err.Error())
			break
		}
		fmt.Printf("%d.%d.%d.%d bytes from %s: icmp_seq=0 ttl=%d time=%d ms", buf[12], buf[13], buf[14], buf[15], 0, buf[8])
		time.Sleep(time.Second)
	}

}

func getChecksum(data []byte) uint16 {
	length := len(data)
	index := 0
	var sum uint32
	// 相邻的字节拼接再求和
	for length > 1 {
		// [16个0 00000011 8个0] + [24个0 00000111] = [16个0 00000011 00000111]
		sum += uint32(data[index]<<8) + uint32(data[index+1])
		length -= 2
		index += 2
	}
	if length == 1 {
		sum += uint32(data[index])
	}
	// 高16位(data >> 16)和低16位(uint16(data))相加
	h := sum >> 16
	for h != 0 {
		sum = h + uint32(uint16(sum))
		h = sum >> 16
	}
	return uint16(^sum)
}
