# reverseconn

This is a test project for reverse proxy with TCP tunnel.

Server will listen to 2 ports, one is waiting for user, another is for client.
Once client coming in, the tunnel is created.

Then every user's request will proxy to client through the tunnel.

Client will proxy the requests to target web server.


## Dec 13, 2023

Current senario is working but the bottleneck is tunnel.
It is impossible to handle throw every request within ONE tunnel since requests are concurrent.

I refered to other proxy package like "tcpproxy", every connection comes in will create a new corresponding connection to do the io.Copy which is more reasonable.
But in order to create as much as the amount of tunnel, server has to allow the ports, I think this may be not a good idea.