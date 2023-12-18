package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

func proxyTest1(src net.Conn) {
	for {
		buffer := make([]byte, 1024)
		n, err := src.Read(buffer)
		if err != nil {
			continue
		}
		buffer = buffer[:n]
		fmt.Println("read from conn:\n", string(buffer))
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
		if err != nil {
			panic(err)
		}
		request.Host = "localhost:9000"

		dst, err := net.Dial("tcp", "localhost:9000")
		if err != nil {
			panic(err)
		}
		fmt.Println("Connected to localhost:9000")

		// Write the modified request to webConn
		err = request.Write(dst)
		if err != nil {
			panic(err)
		}
		fmt.Println("write to webConn:\n", request)

		// read from webConn
		ch := make(chan bool)
		go func() {
			_, err := dst.Read(buffer)
			if err != nil {
				return
			}
			fmt.Println("read from webConn:\n", string(buffer))

			// send to conn
			_, err = src.Write(buffer)
			if err != nil {
				panic(err)
			}
			fmt.Println("write to conn:\n", string(buffer))
			ch <- true
		}()

		//wait
		<-ch
		// dst.Close()
	}
}

func proxyTest2(src net.Conn) {
	for {
		dst, err := net.Dial("tcp", "localhost:9000")
		if err != nil {
			panic(err)
		}
		fmt.Println("Connected to localhost:9000")
		// modify request, request usually is http request and won't be too large
		buffer := make([]byte, 1024)
		n, err := src.Read(buffer)
		if err != nil {
			panic(err)
		}
		fmt.Println("read from conn:", n)
		request, err := http.ReadRequest(bufio.NewReader(bytes.NewReader(buffer)))
		if err != nil {
			panic(err)
		}
		request.Host = "localhost:9000"
		// Write the modified request to webConn
		err = request.Write(dst)
		if err != nil {
			panic(err)
		}
		fmt.Println("write to webConn:\n", request)

		// n64, err := io.Copy(src, dst)
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Println("copied", n64, "to src")
		resp, err := http.ReadResponse(bufio.NewReader(dst), nil)
		if err != nil {
			panic(err)
		}
		fmt.Println("read from dst:\n", resp)
		resp.Write(src)

		dst.Close()
	}
}

// SocketClient connect to server:48000 and keep connection alive
// Usually, this is a client program on a different machine such as a switch, router or embedded device
// When receive data from server, dial to client web server and copy buffer from conn to clientConn and copy buffer from clientConn to conn
func SocketClient() {
	server := "122.147.151.234:27188"
	conn, err := net.Dial("tcp", server)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Println("Connected to", server)

	proxyTest2(conn)
}

func proxyConnection(dst net.Conn, src net.Conn) {
	chDone := make(chan error, 1)
	// Start remote -> local data transfer
	go func() {
		// n, err := io.Copy(dst, src)
		// if err != nil {
		// 	fmt.Println("error while copy remote->local:", err)
		// }
		// fmt.Println("copied", n, "to dst")
		req, err := http.ReadRequest(bufio.NewReader(src))
		if err != nil {
			panic(err)
		}
		fmt.Println("read from src:\n", req)
		req.Write(dst)
		chDone <- err
	}()

	// Start local -> remote data transfer
	go func() {
		// n, err := io.Copy(src, dst)
		// if err != nil {
		// 	fmt.Println("error while copy local->remote:", err)
		// }
		// fmt.Println("copied", n, "to src")
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

func test4(server, proxyTo string) {
	cert, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		fmt.Println("Error loading certificate:", err)
		return
	}

	// Perform SSL/TLS handshake
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true, // Skip certificate verification (for simplicity; not recommended in a production environment)
	}

	tunnel, err := tls.Dial("tcp", server, tlsConfig)
	// tunnel, err := net.Dial("tcp", server)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to", server)

	localWeb, err := tls.Dial("tcp", proxyTo, tlsConfig)
	if err != nil {
		panic(err)
	}
	fmt.Println("Connected to", proxyTo)
	defer localWeb.Close()

	for {
		buffer := make([]byte, 1024)
		n, err := tunnel.Read(buffer)
		if err != nil {
			continue
		}
		fmt.Println("read", n, "from tunnel")
		buffer = buffer[:n]
		fmt.Println(string(buffer))

		n, err = localWeb.Write(buffer)
		if err != nil {
			panic(err)
		}
		fmt.Println("write", n, "to localWeb")

		// var resp []byte
		// var contentLength int = 0
		// var count int = 0
		for {
			buffer = make([]byte, 1024)
			// n, err := localWeb.Read(buffer)
			// read timeout
			localWeb.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, err := localWeb.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					fmt.Println("Read timeout reached")
					break
				}
				continue
			}
			fmt.Println("read", n, "from dst")
			if n == 0 || err == io.EOF {
				break
			}
			buffer = buffer[:n]
			fmt.Println(string(buffer))
			// resp = append(resp, buffer[:n]...)

			n, err = tunnel.Write(buffer)
			if err != nil {
				panic(err)
			}
			fmt.Println("write", n, "to tunnel")

			// sum up the length of response
			// if contentLength > 0 {
			// 	count += n
			// 	fmt.Println("count", count)
			// }

			// Content-Length: xxx
			// if strings.Contains(string(buffer), "Content-Length") {
			// 	s := strings.Split(string(buffer), ":")[1]
			// 	s = strings.TrimSpace(s)
			// 	contentLength, err = strconv.Atoi(s)
			// 	if err != nil {
			// 		panic(err)
			// 	}
			// }

			// if the length of response is equal to Content-Length, break
			// if count == contentLength && contentLength != 0 {
			// 	break
			// }
		}
		// fmt.Println("read", len(resp), "from localWeb")
		// fmt.Println(string(resp))

		// n, err = tunnel.Write(resp)
		// if err != nil {
		// 	panic(err)
		// }
		// fmt.Println("write", n, "to tunnel")

		// localWeb.Close()
	}
}

func test3() {
	// go SocketServer()
	// time.Sleep(1 * time.Second)
	SocketClient()

	// go TcpServer()
}
