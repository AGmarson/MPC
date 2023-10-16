# Peer-to-peer template

A simple template for a project with a peer-to-peer architecture. In the system, each node is able to send a ping to all other nodes in the system, and in return get a response from each node, telling how many times it has pinged that specific node. As each node always sends a ping to all nodes at the same time, the response will always be the same from all nodes.

## How to run

This distributed system has a hardcoded amount of nodes = 3. To run the system, you must run three separate processes with the commands:

`go run . 0`

`go run . 1`

`go run . 2`

## Notes about the main.go file

Each node has to act as server and client at the same time. In the `main()` function, we both setup the node as a server, listening to its own port (lines 30-42), and also set it up as a client, connecting to the other peers as servers (lines 44-60). Both of these are necessary for the whole architecture to work. 

### Changing the ports

In this example, our three nodes run on ports 5000, 5001 and 5002 - this is decided on lines 17-18 and in the "How to run" section. Changing the arguments in the commands, i.e. from `go run . 0` to `go run . 20`, will change the port from 5000 to 5020. Changing the last part of line 18 from `5000` to `3000` will change the port in the same way. 

Changing the ports will affect the "find the other peers" part of the `main()` function. If you make changes to which port each node runs on, make sure to change line 45 so that it matches your new ports.

### Changing the amount of nodes

Line 44 starts a for loop which runs three times, since there are three nodes in the system. Increasing the number to e.g. 5 will enable you to have 5 nodes in the system, and you would simply need to start two more processes with

`go run . 3`

`go run . 4`

### Changing the functionality

You will need a function for each rpc - such that the "server-side" of the node knows what to do, when another node tries to communicate with it. In this example, the `Ping(...)` function is the rpc function. 

You do not necessarily need a function for the "client-side" of the node, but there is some clean-code-good-practice in doing it as such. It is up to you though! In this example, we have a `sendPingToAll()` function, where the peer acts as a client, sends a ping to all the peers, it has connected to, and then prints the replies it gets.

The main functionality is in lines 62-65. This is where we decide what the peer should do. In this example, the peer waits for a newline character to be entered and then calls the `sendPingToAll()` function. This can be changed, such that it calls it every 10 seconds, or such that it does something completely different. 