package gcache

/*
import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

const (
	MODE_HTTP = "http"
	MODE_TCP  = "tcp"
	MODE_UDP  = "udp"
	MODE_TLS  = "tls"

//	MODE_TLS  = "https"
)

type Config struct {
	Mode        string
	BindAddress string
	Expiration  int
	HashFunc    string
}

func main() {
	c := Config{}
	c.initFlags()
	cache := newCacheWithJanitor(time.Duration(c.Expiration)*time.Second, time.Duration(c.Expiration/10)*time.Second)
	cache.Add("a", []byte("am"), 0) //TODO delete this line
	switch c.Mode {
	case MODE_HTTP:
		err := http.ListenAndServe(c.BindAddress, nil)
		if err != nil {
			fmt.Printf("Error: %v", err)
		}
	case MODE_TCP:
		ln, err := net.Listen("tcp", c.BindAddress)
		if err != nil {
			//TODO handle error
		}
		for {
			conn, err := ln.Accept()
			if err != nil {
				//TODO handle error
			}
			go handleTCPConnection(conn, cache)
		}

	case MODE_UDP:
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
	flag.StringVar(&c.Mode, "http", "http", "mode of cachec server: can be "+MODE_HTTP+" "+MODE_TCP+" or "+MODE_UDP)
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
	if c.Mode == MODE_HTTP || c.Mode == MODE_TCP || c.Mode == MODE_UDP {
		return nil
	}
	return fmt.Errorf("Wrong mode: %s", c.Mode)
}

//TODO rebuild to get set command
func handleTCPConnection(conn net.Conn, cache *Cache) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 1024)
	// Read the incoming connection into the buffer.
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	// Send a response back to person contacting us.
	conn.Write([]byte("Message received."))
	// Close the connection when you're done with it.
	conn.Close()
}
*/
