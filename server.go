package gcache

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/golang/protobuf/proto"
)

const (
	modeHTTP = "http"
	modeTCP  = "tcp"
	modeUDP  = "udp"
	modeTLS  = "tls"
	amulet   = "~|~"

//	MODE_TLS  = "https"
)

type Config struct {
	Mode        string
	BindAddress string
	Expiration  int
	HashFunc    string
}

func NewCacheServer() {
	c := Config{}
	c.initFlags()
	cache := NewRwCache(40000, DefaultExpiration, false)
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

func (c *Config) initFlags() {
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

func (c *Config) checkFlags() error {
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
		var t = &TransportItem{}
		data, err := ioutil.ReadAll(conn)
		err = proto.Unmarshal(data, t)

		if err != nil {
			return
		}

		switch t.Command {
		case TransportItem_SET:
			cache.SetOrUpdate(t.GetName(), t.GetObject(), time.Duration(t.GetExpiration()))
		case TransportItem_GET:
			data := cache.Get(t.GetName())
			tr := &TransportItem{
				Name:    t.GetName(),
				Object:  data,
				Command: TransportItem_SET,
			}

			message, _ := proto.Marshal(tr)
			conn.Write(message)
		case TransportItem_PURGE:
			cache.Purge()
		case TransportItem_DEAD:
			cache.Dead()
			return
		}

	}
}
