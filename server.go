package gcache

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	modeHTTP         = "http"
	modeTCPLong      = "tcp_long"
	modeTCPShort     = "tcp_short"
	modeUDP          = "udp"
	systemBufferSize = 1e6 //1Mb
	maxPacketSize    = 1e5 //10Kb
	//	modeTLS  = "tls"
	//	MODE_TLS  = "https"
	amulet              = byte(30) //ANCI Record separator
	closendErrorMessage = "closed network connection"
)

var errDead = errors.New("Dead state")

//TODO remove protobuf write custom Marshal demarhsal
type ServerConfig struct {
	Mode        string
	BindAddress string
	Expiration  int
	HashFunc    string
}
type TCPHandler func(*net.TCPListener, Cacher)

func NewCacheServer() {
	c := ServerConfig{}
	c.initFlags()
	cache := NewRwCache(nil) //TODO
	switch c.Mode {
	case modeHTTP:
		err := http.ListenAndServe(c.BindAddress, nil)
		if err != nil {
			log.Fatalln("Could not bind address: " + c.BindAddress + " error: " + err.Error())
		}
	case modeTCPLong, modeTCPShort:
		tcpAddr, err := net.ResolveTCPAddr("tcp", c.BindAddress)
		if err != nil {
			log.Fatalln("Could not resolve address: " + c.BindAddress + " error: " + err.Error())
		}
		ln, err := net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			log.Fatalln("Could not bind address: " + c.BindAddress + " error: " + err.Error())
		}
		if c.Mode == modeTCPLong {
			handleLongTCP(ln, cache)
		} else {
			handleShortTCP(ln, cache)
		}

	case modeUDP:
		udpAdd, err := net.ResolveUDPAddr("udp", c.BindAddress)
		if err != nil {
			log.Fatalln("Could not resolve address: " + c.BindAddress + " error: " + err.Error())
		}
		conn, err := net.ListenUDP("udp", udpAdd)
		if err != nil {
			log.Fatalln("Could not resolve address: " + c.BindAddress + " error: " + err.Error())
		}
		HandleUDP(conn, cache)

	default:
		panic("Not implemented mode : " + c.Mode)
	}

}

func (c *ServerConfig) initFlags() {
	flag.StringVar(&c.Mode, "http", "http", "mode of cachec server: can be "+modeHTTP+" "+modeTCPLong+" or "+modeUDP)
	flag.StringVar(&c.BindAddress, "bind", "", "optional options to set listening specific interface: <ip ro hostname>:<port>")
	flag.IntVar(&c.Expiration, "expiration", 200, "expiration time in seconds")

	flag.Parse()

	if err := c.checkFlags(); err != nil {
		fmt.Printf("Error: %v", err)
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func (c *ServerConfig) checkFlags() error {
	if c.Mode == modeHTTP || c.Mode == modeTCPLong || c.Mode == modeUDP {
		return nil
	}
	return fmt.Errorf("Wrong mode: %s", c.Mode)
}

//handleShortTCP expect only one message via tcp and return data for each
func handleShortTCP(ln *net.TCPListener, cache Cacher) {
	var (
		once    sync.Once
		income  = make(chan *net.TCPConn, 10)
		wg      sync.WaitGroup
		stopper = func() {
			close(income)
			ln.Close()
		}
	)
	handler := func(inCon <-chan *net.TCPConn) {
		defer wg.Done()
		for c := range inCon {
			data, err := ioutil.ReadAll(c) //End client should close write tcp we wait eof

			var t *ItemMessage
			err = proto.Unmarshal(data, t)
			if err != nil {
				c.Close()
				continue
			}
			result, err := handleRequest(t, cache)
			if err == errDead {
				c.Close()
				once.Do(stopper)
				return
			}
			if len(result) > 0 {
				c.Write(result) //Do not care about error we can't do anything with error
			}
			c.Close()
		}
	}

	wg.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go handler(income)
	}

	for {
		conn, err := ln.AcceptTCP()

		if err != nil {
			errStr := err.Error()
			//How to check closed conn better ?
			if strings.Contains(errStr, closendErrorMessage) {
				once.Do(stopper)
				break
			}
			if conn != nil {
				conn.Close() // some error appear do not try to handle income request
			}
			continue
		}
		income <- conn
	}

	wg.Wait()
}

//handleLongTCP expecte open connection and communicate without closing of this
func handleLongTCP(ln *net.TCPListener, cache Cacher) {
	var (
		once sync.Once

		stopper = func() {
			ln.Close()
		}
	)
	handler := func(c *net.TCPConn) {
		buf := make([]byte, maxPacketSize, maxPacketSize)
		for eof := false; eof; {
			n, err := c.Read(buf)
			if err != nil {
				if err == io.EOF {
					eof = true
					break
				}
			}
			if n == 0 {
				continue //Empty request
			}
			var t = &ItemMessage{} //TODO pool messages protobuf sucs
			err = proto.Unmarshal(buf[:n], t)
			if err != nil {
				continue
			}
			result, err := handleRequest(t, cache)
			if err == errDead {
				once.Do(stopper)
				break
			}

			if len(result) > 0 {
				c.Write(result) //Do not care about error we can't do anything with error
			}
		}
		c.Close()

	}

	for {
		conn, err := ln.AcceptTCP()

		if err != nil {
			errStr := err.Error()
			//How to check closed conn better ?
			if strings.Contains(errStr, closendErrorMessage) {
				once.Do(stopper)
				break
			}
			if conn != nil {
				conn.Close() // some error appear do not try to handle income request
			}
			continue
		}
		go handler(conn) //Long handler
	}

}

//HandleUDP is simple handeler with worker pool, we have limitation with updpacket size to bufSize
func HandleUDP(ServerConn *net.UDPConn, cache Cacher) error {

	var (
		once    sync.Once
		wg      sync.WaitGroup
		stopper = func() {
			ServerConn.Close()
		}
	)

	defer once.Do(stopper)
	ServerConn.SetReadBuffer(systemBufferSize)
	handler := func() {
		var (
			buf  = make([]byte, maxPacketSize, maxPacketSize) //TODO pool ?
			n    int
			addr *net.UDPAddr
			err  error
		)
	mainLoop:
		for {
			n, addr, err = ServerConn.ReadFromUDP(buf)
			if err != nil {
				errStr := err.Error()
				//How to check closed conn better ?
				if strings.Contains(errStr, closendErrorMessage) {
					break mainLoop
				}
			}

			var t *ItemMessage
			err = proto.Unmarshal(buf[0:n], t)
			if err != nil {
				continue
			}
			responce, err := handleRequest(t, cache)
			if len(responce) > 0 {
				ServerConn.WriteToUDP(responce, addr)
			}
			if err == errDead {
				once.Do(stopper)
				break mainLoop
			}
		}
		wg.Done()
	}

	wg.Add(runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		go handler() //a little faster with multiple UDP packet
	}

	wg.Wait()

	return nil
}

func handleRequest(t *ItemMessage, cache Cacher) (message []byte, err error) {
	var (
		name   = t.GetName()
		object = t.GetObject()
	)
	switch t.Command {
	case ItemMessage_SET:
		cache.SetOrUpdate(name, object, time.Duration(t.GetExpiration()))
	case ItemMessage_GET:
		data := cache.Get(name)
		tr := &ItemMessage{
			Name:    name,
			Object:  data,
			Command: ItemMessage_SET,
		}
		message, err = proto.Marshal(tr)
		return message, err
	case ItemMessage_PURGE:
		cache.Purge()
	case ItemMessage_DEAD:
		cache.Dead()
		return nil, errDead
	}
	return nil, nil
}
