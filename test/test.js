var assert = require('assert');
var server = require("../index.js")
let WebSocketClient = require('websocket').client
const ganache = require("ganache-cli");
const ganache_server1 = ganache.server();
const ganache_server2 = ganache.server();

var socket;

before(async () => {
    // Setup the eth websockets server that is being used
    await ganache_server1.listen("8545", "localhost")

    await ganache_server2.listen("7545", "localhost")
})

beforeEach(async () => {
    // Setup the failover server to be used for all tests, passing in a local instance of a blockchain client
    // Change second argument to true to log from the server during tests
    server.start(["ws://localhost:8545/", "ws://localhost:7545/"], false)

    // Setup a websocket to the server
    socket = new WebSocketClient()
    socket.connect("ws://localhost:4000/")

})


afterEach((done) => {
    socket.abort()
    server.close()
    done()
})

after((done) => {
    ganache_server1.close()
    ganache_server2.close()
    done()
})


it("can connect to the failover server", (done) => {
    socket.on('connectFailed', (error) => {
        console.log('Connect Error: ' + error.toString());
        assert.fail("Could not connect to server")
    });

    socket.on('connect', (connection) => {
        done()
        connection.close()
    });
});

it("can connect to the ETH client", (done) => {

    socket.on('connectFailed', (error) => {
        assert.fail("Could not connect to server")
    });

    socket.on('connect', (connection) => {

        // Send message to ETH client for a response
        connection.send('{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}')

        connection.on("message", (message) => {
            // Response from ETH Client recieved
            done()
            connection.close()
        })

        connection.on("error", (err) =>{
            console.log("err: " + err)
            assert.fail("Error in response from ETH client")
        })
    });

});

it("can failover to a second ETH client", (done) => {
    ganache_server1.close(function () {
        socket.on('connect', (connection) => {
            connection.on("close", () => {
                // Reconnect and now use backup ganache server
                socket.connect("ws://localhost:4000/")
                socket.on('connect', (_connection) => {
                    // Resend the message
                    _connection.send('{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}')

                    _connection.on("message", (message) => {
                        // Response from ETH Client recieved
                        done()
                        _connection.close()
                    })
                });
            })
        })
    })
});

it("connection is closed when no clients are available", (done) => {
    ganache_server2.close(function () {
        socket.on("connect", (connection) => {
            connection.send('{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}')

            connection.on("message", (msg) => {
                // Should not recieve a message from the ETH client
                assert.fail()
            })

            connection.on("close", (reasonCode, description) => {
                // Connection closed due to no ETH clients being available
                done()
            })
        })
    })
});