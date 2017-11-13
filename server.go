package gcache

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
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
	//cache := NewRwCache(10, time.Duration(c.Expiration*time.Second), false)
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
		for {
			conn, err := ln.Accept()
			if err != nil {
				//TODO handle error
			}
			conn.Close()
			//go handleTCPConnection(conn, cacher)
		}

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
func handleTCPConnection(conn net.Conn, cache Cacher) {
	z := bufio.NewReader(conn)
	v, err := z.ReadBytes(byte('\n'))
	if err != nil {
		return
	}
	data := bytes.SplitN(v, []byte(amulet), 2)
	if len(data) != 2 {
		return
	}
	cache.SetOrUpdate(string(data[0]), data[1], 0)
}
