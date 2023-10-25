# How to run

If you are on windows you should be able to run the batch file `run.bat` which will open 4 terminals for you.

Otherwise you can run 4 different terminals manually with one command per terminal:

`go run . 0`

`go run . 1`

`go run . 2`

`go run . 3`

## The program explained and how to use once running

**The first terminal** where the command `go run . 0` was run is the "hosptial".
This will receive data from the other peers and broadcast results. It does not take input from the user.

**The three other terminals** are the clients.
In order to compute a result you must enter exaclty one number in each client terminal.
This will cause them to break the number into chunks and share them among the clients.
Once each client has 3 chunks they will share the sum with the hospital terminal which will then compute the total sum and broadcast it.

**Do not** enter more than one number into each client until a result has been broadcasted. This may break the protocol.
