package spider

import (
	"bytes"
	"container/list"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type ICMP struct {
	Type        uint8
	Code        uint8
	Checksum    uint16
	Identifier  uint16
	SequenceNum uint16
}

type PingResult struct {
	Dst      string
	Sended   int
	Recved   int
	Losted   int
	Min      int64
	Max      int64
	LostRate float32
	Average  float32
}

func Telnet(dst string) error {
	conn, err := net.Dial("tcp", dst)
	if err != nil {
		return err
	}
	defer conn.Close()
	return nil
}

func Ping(ret *PingResult, host, port string) error {
	var icmp ICMP
	terr := Telnet(fmt.Sprintf("%s:%s", host, port))
	if terr != nil {
		return terr
	}
	conn, err := net.Dial("ip4:icmp", host)
	if err != nil {
		return err
	}
	defer conn.Close()

	icmp.Type = 8
	icmp.Code = 0
	icmp.Checksum = 0
	icmp.Identifier = 0
	icmp.SequenceNum = 0

	var buffer bytes.Buffer
	binary.Write(&buffer, binary.BigEndian, icmp)
	icmp.Checksum = CheckSum(buffer.Bytes())
	buffer.Reset()
	binary.Write(&buffer, binary.BigEndian, icmp)

	recv := make([]byte, 1024)
	statistic := list.New()
	sended_packets := 0

	for i := 4; i > 0; i-- {
		if _, err := conn.Write(buffer.Bytes()); err != nil {
			continue
		}
		sended_packets++
		t_start := time.Now()

		conn.SetReadDeadline((time.Now().Add(time.Second * 5)))
		_, err := conn.Read(recv)

		if err != nil {
			continue
		}
		t_end := time.Now()
		dur := t_end.Sub(t_start).Nanoseconds() / 1e6
		statistic.PushBack(dur)
	}

	var min, max, sum int64
	if statistic.Len() == 0 {
		min, max, sum = 0, 0, 0
	} else {
		min, max, sum = statistic.Front().Value.(int64), statistic.Front().Value.(int64), int64(0)
	}

	for v := statistic.Front(); v != nil; v = v.Next() {
		val := v.Value.(int64)
		switch {
		case val < min:
			min = val
		case val > max:
			max = val
		}
		sum = sum + val
	}
	recved, losted := statistic.Len(), sended_packets-statistic.Len()

	ret.Dst = host
	ret.Sended = sended_packets
	ret.Recved = recved
	ret.Losted = losted
	ret.Min = min
	ret.Max = max
	ret.LostRate = float32(losted) / float32(sended_packets) * 100
	ret.Average = float32(sum) / float32(recved)
	return nil
}

func CheckSum(data []byte) uint16 {
	var (
		sum    uint32
		length int = len(data)
		index  int
	)
	for length > 1 {
		sum += uint32(data[index])<<8 + uint32(data[index+1])
		index += 2
		length -= 2
	}
	if length > 0 {
		sum += uint32(data[index])
	}
	sum += (sum >> 16)
	return uint16(^sum)
}
