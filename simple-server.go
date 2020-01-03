package main

import (
	"fmt"
	"log"
	"net"
)

func main() {
	pc, err := net.ListenPacket("udp", ":1234")
	if err != nil {
		log.Fatal(err)
		return
	}

	// `Close`ing the packet "connection" means cleaning the data structures
	// allocated for holding information about the listening socket.
	defer pc.Close()

	for {
		fmt.Println("Listening!")
		buf := make([]byte, 1024)
		n, addr, err := pc.ReadFrom(buf)
		if err != nil {
			continue
		}

		go serve(pc, addr, buf[:n])
	}
}

func serve(pc net.PacketConn, addr net.Addr, buf []byte) {
	// 0 - 1: ID
	// 2: QR(1): Opcode(4)
	buf[2] |= 0x80 // Set QR bit

	pc.WriteTo(buf, addr)
}
