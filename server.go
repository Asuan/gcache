package gcache

import (
	"errors"
	"flag"
	"fmt"
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
	modeHTTP = "http"
	modeTCP  = "tcp"
	modeUDP  = "udp"

//	modeTLS  = "tls"
//	MODE_TLS  = "https"
)

var errDead = errors.New("Dead state")

type ServerConfig struct {
	Mode        string
	BindAddress string
	Expiration  int
	HashFunc    string
}

func NewCacheServer() {
	c := ServerConfig{}
	c.initFlags()
	cache := NewRwCache(nil) //TODO
	switch c.Mode {
	case modeHTTP:
		err := http.ListenAndServe(c.BindAddress, nil)
		if err != nil {
			fmt.Printf("Error: %v", err)
		}
	case modeTCP:
		ln, err := net.Listen("tcp", c.BindAddress)
		if err != nil {
			//TODO handle error
		}
		handleTCPConnection(ln, cache)

	case modeUDP:
		udpAdd, err := net.ResolveUDPAddr("", c.BindAddress)
		if err != nil {
			log.Fatalln("Could not resolve address: " + c.BindAddress)
		}
		lnu, err := net.ListenUDP("udp", udpAdd)
		if err != nil {
			//TODO handle error
		}
		for {
			buf := make([]byte, 8192) //TODO parametrize it 8k
			_, _, err := lnu.ReadFromUDP(buf)
			//TODO hande incoming message.

			if err != nil {
				log.Fatal(err)
			}
		}

	default:
		panic("Not implemented mode : " + c.Mode)
	}

}

func (c *ServerConfig) initFlags() {
	flag.StringVar(&c.Mode, "http", "http", "mode of cachec server: can be "+modeHTTP+" "+modeTCP+" or "+modeUDP)
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
	if c.Mode == modeHTTP || c.Mode == modeTCP || c.Mode == modeUDP {
		return nil
	}
	return fmt.Errorf("Wrong mode: %s", c.Mode)
}

//TODO rebuild to async reader writer
func handleTCPConnection(ln net.Listener, cache Cacher) {

	conn, err := ln.Accept()
	if err != nil {
		if conn != nil {
			conn.Close() //
		}
	}
	for {
		//var buf bytes.Buffer
		//io.Copy(&buf, conn)
		var t = &ItemMessage{}
		data, err := ioutil.ReadAll(conn)
		err = proto.Unmarshal(data, t)

		if err != nil {
			return
		}

	}
}

//HandleUDP is
func HandleUDP(addr *net.UDPAddr, cache Cacher) error {
	const bufSize = 1500 //Depend on MTU
	var once sync.Once
	var wg sync.WaitGroup

	ServerConn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	stopper := func() {
		ServerConn.Close()
	}
	defer once.Do(stopper)

	handler := func() {
		var (
			buf  = make([]byte, bufSize, bufSize)
			n    int
			addr *net.UDPAddr
			err  error
		)
	mainLoop:
		for {
			n, addr, err = ServerConn.ReadFromUDP(buf)
			if err != nil {
				errStr := err.Error()
				//Huh to check closed stte better ?
				if strings.HasSuffix(errStr, "closed network connection") {
					break mainLoop
				}
			}

			var t *ItemMessage
			err = proto.Unmarshal(buf[0:n], t)
			if err != nil {
				continue
			}
			responce, err := handleCommand(t, cache)
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
		go handler() //little faster
	}

	wg.Wait()
	return nil
}

func handleCommand(t *ItemMessage, cache Cacher) (message []byte, err error) {
	switch t.Command {
	case ItemMessage_SET:
		cache.SetOrUpdate(t.GetName(), t.GetObject(), time.Duration(t.GetExpiration()))
	case ItemMessage_GET:
		data := cache.Get(t.GetName())
		tr := &ItemMessage{
			Name:    t.GetName(),
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
