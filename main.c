#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <sys/epoll.h>

#define BUFFER_SIZE 1024

int webConnection()
{
    // Dial to device web server
    int webConn = socket(AF_INET, SOCK_STREAM, 0);
    if (webConn == -1)
    {
        perror("Error creating webConn socket");
        // exit(EXIT_FAILURE);
        return -1;
    }

    struct sockaddr_in webServerAddr;
    memset(&webServerAddr, 0, sizeof(webServerAddr));
    webServerAddr.sin_family = AF_INET;
    webServerAddr.sin_addr.s_addr = inet_addr("127.0.0.1");
    webServerAddr.sin_port = htons(443);

    if (connect(webConn, (struct sockaddr *)&webServerAddr, sizeof(webServerAddr)) == -1)
    {
        perror("Error connecting to device web server");
        // exit(EXIT_FAILURE);
        return -1;
    }

    return webConn;
}

bool testWebConnection()
{
    int webConn = webConnection();
    if (webConn == -1)
    {
        return false;
    }

    char *request = "GET / HTTP/1.1\r\nHost: 127.0.0.1\r\n\r\n";
    if (send(webConn, request, strlen(request), 0) == -1)
    {
        perror("Error sending request to web server");
        return false;
    }

    char buffer[1024];
    int len = recv(webConn, buffer, sizeof(buffer) - 1, 0);
    if (len == -1)
    {
        perror("Error receiving response from web server");
        return false;
    }

    buffer[len] = '\0';
    printf("Received response from web server: %s\n", buffer);

    close(webConn);
    return true;
}

int main()
{
    if (!testWebConnection())
    {
        printf("Error connecting to web server\n");
        exit(EXIT_FAILURE);
    }

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
    serverAddr.sin_port = htons(27188);

    if (connect(conn, (struct sockaddr *)&serverAddr, sizeof(serverAddr)) == -1)
    {
        perror("Error connecting to server");
        exit(EXIT_FAILURE);
    }

    printf("Connected to server\n");

    int epoll_fd = epoll_create(1);
    if (epoll_fd == -1)
    {
        perror("Error creating epoll");
        exit(EXIT_FAILURE);
    }
    struct epoll_event event;
    event.events = EPOLLIN;
    event.data.fd = conn;
    if (epoll_ctl(epoll_fd, EPOLL_CTL_ADD, conn, &event) == -1)
    {
        perror("Error adding conn to epoll");
        exit(EXIT_FAILURE);
    }

    while (1)
    {
        struct epoll_event events[1];
        int n = epoll_wait(epoll_fd, events, 1, -1);
        if (n == -1)
        {
            perror("Error waiting for epoll events");
            exit(EXIT_FAILURE);
        }

        for (int i = 0; i < n; i++)
        {
            if (events[i].data.fd == conn)
            {
                char buffer[BUFFER_SIZE];
                int len = recv(conn, buffer, BUFFER_SIZE, 0);
                if (len == -1)
                {
                    perror("Error reading from conn");
                    exit(EXIT_FAILURE);
                }
                if (len == 0)
                {
                    printf("Connection closed by server\n");
                    exit(EXIT_SUCCESS);
                }
                printf("Received %d bytes from server\n", len);

                int webConn = webConnection();
                if (webConn == -1)
                {
                    printf("Error connecting to web server\n");
                    continue;
                }
                printf("Connected to web server\n");

                if (send(webConn, buffer, len, 0) == -1)
                {
                    perror("Error sending to web server");
                    exit(EXIT_FAILURE);
                }

                printf("Sent %d bytes to web server\n", len);

                while (1)
                {
                    len = recv(webConn, buffer, BUFFER_SIZE, 0);
                    if (len == -1)
                    {
                        perror("Error reading from web server");
                        exit(EXIT_FAILURE);
                    }
                    if (len == 0)
                    {
                        printf("Connection closed by web server\n");
                        // exit(EXIT_SUCCESS);
                        break;
                    }
                    printf("Received %d bytes from web server\n", len);

                    if (send(conn, buffer, len, 0) == -1)
                    {
                        perror("Error sending to conn");
                        exit(EXIT_FAILURE);
                    }
                    printf("Sent %d bytes to server\n", len);
                }
                close(webConn);
            }
        }
    }

    return 0;
}