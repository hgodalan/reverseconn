#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <arpa/inet.h>
#include <sys/wait.h>

#define BUFFER_SIZE 1024

void proxyConnection(int dst, int src)
{
    // Start remote -> local data transfer
    pid_t remoteToLocal = fork();
    if (remoteToLocal == 0)
    {
        char buffer[BUFFER_SIZE];
        ssize_t bytesRead;

        while ((bytesRead = read(src, buffer, sizeof(buffer))) > 0)
        {
            write(dst, buffer, bytesRead);
        }

        close(src);
        close(dst);
        exit(EXIT_SUCCESS);
    }
    else if (remoteToLocal == -1)
    {
        perror("Error creating remoteToLocal process");
        exit(EXIT_FAILURE);
    }

    // Start local -> remote data transfer
    pid_t localToRemote = fork();
    if (localToRemote == 0)
    {
        char buffer[BUFFER_SIZE];
        ssize_t bytesRead;

        while ((bytesRead = read(dst, buffer, sizeof(buffer))) > 0)
        {
            write(src, buffer, bytesRead);
        }

        close(src);
        close(dst);
        exit(EXIT_SUCCESS);
    }
    else if (localToRemote == -1)
    {
        perror("Error creating localToRemote process");
        exit(EXIT_FAILURE);
    }

    // Wait for child processes to finish
    int status;
    waitpid(remoteToLocal, &status, 0);
    waitpid(localToRemote, &status, 0);
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

    printf("Connected to server\n");

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

    // Continuous proxy handling
    while (1)
    {
        proxyConnection(conn, webConn);
    }

    return 0;
}
