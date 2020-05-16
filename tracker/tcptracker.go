package tracker

import (
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

type TCPTracker struct {
	Host     string
	conn     *net.UDPConn
	InfoHash [20]byte
	Length   uint64
}

func (tracker *TCPTracker) Connect() (e error) {
	response, err := http.Get("http:/tracker.trackerfix.com/announce")
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	} else {
		defer response.Body.Close()
		contents, err := ioutil.ReadAll(response.Body)
		if err != nil {
			fmt.Printf("%s", err)
			os.Exit(1)
		}
		fmt.Printf("%s\n", string(contents))
	}

	return nil
}
