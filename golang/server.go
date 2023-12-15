// waiting for 2nd proxy which behind the firewall to connect
func SocketServer() {
	serverIP := "0.0.0.0"
	serverPort := 27188

	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		fmt.Println("Error loading certificate:", err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Skip certificate verification (for simplicity; not recommended in a production environment)
	}

	//  tcp server
	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", serverIP, serverPort), tlsConfig)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("socket server listening on 27188...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Printf("Port %d: Accepted connection from %s\n", serverPort, conn.RemoteAddr().String())

		// add the connection to the map , certain port we provide:conn
		ClientConnections[27185] = conn
	}
}

func ReverseConn() {
	go SocketServer()
	time.Sleep(1 * time.Second)

	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		fmt.Println("Error loading certificate:", err)
		return
	}

	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Skip certificate verification (for simplicity; not recommended in a production environment)
	}

	listerner, err := tls.Listen("tcp", ":27185", tlsConfig)
	if err != nil {
		panic(err)
	}
	defer listerner.Close()
	fmt.Println("waiting for requests on 27185...")

	for {
		userConn, err := listerner.Accept()
		if err != nil {
			panic(err)
		}

		clientConn, exists := ClientConnections[27185]
		if !exists {
			fmt.Printf("Client %s is not connected\n", userConn.RemoteAddr().String())
			continue
		}
		proxyConnection2(clientConn, userConn)
		userConn.Close()
	}
}

func proxyConnection2(tunnel net.Conn, userConn net.Conn) {

	// Parse the underlying HTTP request
	request, err := http.ReadRequest(bufio.NewReader(userConn))
	if err != nil {
		fmt.Println("Error reading HTTP request:", err)
		return
	}

	// Modify headers or perform any other proxy logic
	request.Host = "192.168.1.1"
	fmt.Println(request)

	if err := request.Write(tunnel); err != nil {
		fmt.Println("Error writing request to destination server:", err)
		return
	}
	fmt.Println("send to tunnel")

	// Read the response from the destination server
	response, err := http.ReadResponse(bufio.NewReader(tunnel), request)
	if err != nil {
		fmt.Println("Error reading response from destination server:", err)
		return
	}
	fmt.Println(response)

	// Modify headers or perform any other proxy logic
	// ...

	// Send the modified response back to the client
	if err := response.Write(userConn); err != nil {
		fmt.Println("Error writing response to client:", err)
		return
	}
}