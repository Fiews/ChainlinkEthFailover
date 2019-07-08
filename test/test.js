var assert = require('assert');
var server = require("../index.js")
let WebSocketClient = require('websocket').client
const ganache = require("ganache-cli");
const ganache_server1 = ganache.server();
const ganache_server2 = ganache.server();

const sleep = (milliseconds) => {
    return new Promise(resolve => setTimeout(resolve, milliseconds))
}

var socket;

before(async function () {
    //Setup the eth websockets server that is being used
    await ganache_server1.listen("8545", "localhost")

    await ganache_server2.listen("7545", "localhost")
})

beforeEach(async function () {
    //this.timeout(0);
    //Setup the failover server to be used for all tests, passing in a local instance of a blockchain client
    server.start(["ws://localhost:8545/", "ws://localhost:7545/"])

    //Setup a websocket to the server
    socket = new WebSocketClient()
    socket.connect("ws://localhost:4000/")

})


afterEach(function (done) {
    socket.abort()
    server.close()
    done()
})

after(function(done) {
    ganache_server1.close()
    ganache_server2.close()
    done()
})


it("can connect to the failover server", function (done) {
    socket.on('connectFailed', function (error) {
        console.log('Connect Error: ' + error.toString());
        assert.fail("Could not connect to server")
    });

    socket.on('connect', function (connection) {
        done()
        connection.close()
    });
});

it("can connect to the ETH client", function (done) {
    //this.timeout(0)

    socket.on('connectFailed', function (error) {
        assert.fail("Could not connect to server")
    });

    socket.on('connect', function (connection) {

        //Send message to ETH client for a response
        connection.send('{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}')

        connection.on("message", function (message) {
            //Response from ETH Client recieved
            done()
            connection.close()
        })

        connection.on("error", function (err) {
            console.log("err: " + err)
            assert.fail("Error in response from ETH client")
        })
    });

});

it("can failover to a second ETH client", function (done) {
    //this.timeout(0)

    socket.on('connect', function (connection) {

        //Close server 1 and then send a message
        ganache_server1.close(function () {
            console.log("Closed server 1")

            connection.on("close", function () {
                //Reconnect and now use backup ganache server
                socket.connect("ws://localhost:4000/")
                socket.on('connect', function (_connection) {
                    //Resend the message
                    _connection.send('{"jsonrpc":"2.0","method":"eth_accounts","params":[],"id":1}')

                    _connection.on("message", function (message) {
                        //Response from ETH Client recieved
                        done()
                        _connection.close()
                        //Restart ganache server 1
                        //ganache_server1.listen("8545", "localhost")
                    })
                });
            })
        });
    })
});