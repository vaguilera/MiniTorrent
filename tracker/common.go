package tracker

import (
	"bytes"
	"encoding/binary"
	"net"
)

type Peer struct {
	IP     net.IP
	Port   uint16
	Status byte
}

func StructToBuffer(st interface{}) []byte {
	buf := &bytes.Buffer{}
	err := binary.Write(buf, binary.BigEndian, st)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
