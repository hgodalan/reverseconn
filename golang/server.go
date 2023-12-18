package main

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	//"io"
	"crypto/tls"
)

// ClientConnections map[port]net.Conn, server local port:conn
var ClientConnections map[int]net.Conn

func init() {
	ClientConnections = make(map[int]net.Conn)
}

func TunnelServer(userPort, tunnelPort int) {
	serverIP := "0.0.0.0"

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
	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", serverIP, tunnelPort), tlsConfig)
	if err != nil {
		panic(err)
	}
	defer listener.Close()

	fmt.Println("socket server listening on", tunnelPort, "...")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}
		fmt.Printf("Port %d: Accepted connection from %s\n", tunnelPort, conn.RemoteAddr().String())

		// add the connection to the map , certain port we provide:conn
		ClientConnections[userPort] = conn
	}
}

func serverProxy(dst net.Conn, src net.Conn) {
	chDone := make(chan error, 2)
	// Start remote -> local data transfer
	go func() {
		req, err := http.ReadRequest(bufio.NewReader(src))
		if err != nil {
			panic(err)
		}
		fmt.Println("read from src:\n", req)
		req.Host = "localhost:9000"
		req.Write(dst)
		chDone <- err
	}()

	// Start local -> remote data transfer
	go func() {
		br := bufio.NewReader(dst)
		resp, err := http.ReadResponse(br, nil)
		if err != nil {
			panic(err)
		}
		fmt.Println("read from dst:\n", resp)
		resp.Write(src)
		chDone <- err
	}()

	<-chDone
	<-chDone
}

func serverProxy2(tunnel net.Conn, userConn net.Conn) {
	for {
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

		// Check the Connection header in the response
		if response.Header.Get("Connection") == "close" {
			return
		}
	}
}

func ServerRun(userPort, tunnelPort int) {
	go TunnelServer(userPort, tunnelPort)
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

	listerner, err := tls.Listen("tcp", fmt.Sprintf(":%d", userPort), tlsConfig)
	if err != nil {
		panic(err)
	}
	defer listerner.Close()
	fmt.Println("waiting for requests on", userPort, "...")

	for {
		userConn, err := listerner.Accept()
		if err != nil {
			panic(err)
		}

		clientConn, exists := ClientConnections[userPort]
		if !exists {
			fmt.Printf("Client %s is not connected\n", userConn.RemoteAddr().String())
			continue
		}
		//go func() {
		serverProxy2(clientConn, userConn)
		userConn.Close()
		//}()
	}
}
