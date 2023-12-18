package main

import (
	"flag"
	"strconv"
)

func main() {

	// is client or server
	var isClient bool
	flag.BoolVar(&isClient, "c", false, "is client")

	// server port for user to connect
	var userConnectPort int
	flag.IntVar(&userConnectPort, "userp", 27185, "server port for user to connect")
	// server port for client to connect to create tunnel
	var serverTunnelPort int
	flag.IntVar(&serverTunnelPort, "tunp", 27188, "server port for client to connect")

	// client proxy to
	var proxyTo string
	flag.StringVar(&proxyTo, "proxy", "localhost:9000", "client proxy to")
	// server ip or domain
	var serverIp string
	flag.StringVar(&serverIp, "server", "122.147.151.234", "server address")

	flag.Parse()

	if isClient {
		tmp := serverIp + ":" + strconv.Itoa(serverTunnelPort)
		test4(tmp, proxyTo)
	} else {
		ServerRun(userConnectPort, serverTunnelPort)
	}
}
