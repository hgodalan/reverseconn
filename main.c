#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <arpa/inet.h>

/*
 *  This is a simple TCP proxy client example written in Golang.
    *  First, it connects to the TCP server, then it sends the request got from the tunnel.
    *  After that, it reads the response from the server and sends it to the local service which is a web server.
    * Finally, it reads the response from the local web server and sends it back to the tunnel.
    *

 func main() {
    server := "122.147.151.234:27188"
    conn, err := net.Dial("tcp", server)
    if err != nil {
        panic(err)
    }
    defer conn.Close()

    fmt.Println("Connected to", server)

    proxyConnection(conn)
}

func proxyConnection(src net.Conn) {
    for {
        dst, err := net.Dial("tcp", "localhost:443")
        if err != nil {
            panic(err)
        }
        fmt.Println("Connected to localhost:443")
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
        request.Host = "localhost:443"
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
 */

int localWebConn()
{
    // Dial to device web server
    int webConn = socket(AF_INET, SOCK_STREAM, 0);
    if (webConn == -1)
    {
        perror("Error creating webConn socket");
        exit(EXIT_FAILURE);
    }

    struct sockaddr_in webServerAddr;
    memset(&webServerAddr, 0, sizeof(webServerAddr));
    webServerAddr.sin_family = AF_INET;
    webServerAddr.sin_addr.s_addr = inet_addr("127.0.0.1");
    webServerAddr.sin_port = htons(443);

    if (connect(webConn, (struct sockaddr *)&webServerAddr, sizeof(webServerAddr)) == -1)
    {
        perror("Error connecting to device web server");
        exit(EXIT_FAILURE);
    }
}

void proxyConnection(int src)
{
    while (1)
    {
        // ready to read from src
        fd_set read_fds;
        FD_ZERO(&read_fds);
        FD_SET(src, &read_fds);

        struct timeval timeout;
        timeout.tv_sec = 5;
        timeout.tv_usec = 0;

        int select_res = select(src + 1, &read_fds, NULL, NULL, &timeout);
        if (select_res == -1)
        {
            perror("select failed");
            break;
        }
        else if (select_res == 0)
        {
            // Timeout, no data to read
            continue;
        }

        int webConn = localWebConn();
        if (webConn == -1)
        {
            perror("Error connecting to device web server");
            continue;
        }

        // If we get here, there is data to read on src
        char buffer[1024];
        ssize_t len = recv(src, buffer, sizeof(buffer) - 1, 0);
        if (len == -1)
        {
            perror("recv failed");
            break;
        }
        buffer[len] = '\0';

        // Modify request Host to localhost:443
        char *hostHeader = strstr(buffer, "Host:");
        if (hostHeader)
        {
            char *endOfHostHeader = strstr(hostHeader, "\r\n");
            if (endOfHostHeader)
            {
                size_t hostHeaderLength = endOfHostHeader - hostHeader;
                char newHostHeader[hostHeaderLength + 1];
                snprintf(newHostHeader, sizeof(newHostHeader), "Host: localhost:443");
                memcpy(hostHeader, newHostHeader, strlen(newHostHeader));
            }
        }

        // Send the modified request to the web connection
        if (send(webConn, buffer, len, 0) == -1)
        {
            perror("send failed");
            break;
        }

        // Read response from web connection, maybe very large, so we need to read in chunks
        while (1)
        {
            memset(buffer, 0, sizeof(buffer));
            ssize_t len = recv(webConn, buffer, sizeof(buffer) - 1, 0);
            if (len == -1)
            {
                perror("recv failed");
                break;
            }
            else if (len == 0)
            {
                // No more data to read
                break;
            }

            // Send response to src
            if (send(src, buffer, len, 0) == -1)
            {
                perror("send failed");
                break;
            }
        }

        close(webConn);
    }

    return NULL;
}

size_t write_callback(void *contents, size_t size, size_t nmemb, void *userp)
{
    size_t realsize = size * nmemb;
    int src = *(int *)userp;
    char buffer[realsize];
    printf("write_callback: %d bytes\n", realsize);
    memcpy(buffer, contents, realsize);
    write(src, buffer, realsize);

    return realsize;
}

int main()
{
    int conn = socket(AF_INET, SOCK_STREAM, 0);
    if (conn == -1)
    {
        perror("Error creating socket");
        exit(EXIT_FAILURE);
    }

    struct sockaddr_in serverAddr;
    memset(&serverAddr, 0, sizeof(serverAddr));
    serverAddr.sin_family = AF_INET;
    serverAddr.sin_addr.s_addr = inet_addr("122.147.151.234");
    serverAddr.sin_port = htons(48000);

    if (connect(conn, (struct sockaddr *)&serverAddr, sizeof(serverAddr)) == -1)
    {
        perror("Error connecting to server");
        exit(EXIT_FAILURE);
    }

    printf("Connected to tcp server\n");

    proxyConnection(conn);

    return 0;
}
