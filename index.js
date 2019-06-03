let WebSocketServer = require('websocket').server;
let WebSocketClient = require('websocket').client;
let http = require('http');

let endpoints = [];

if (process.argv.length <= 2) {
    console.error('No endpoints provided');
    process.exit();
}

for (let i = 2; i < process.argv.length; i++) {
    console.log('Adding endpoint '+process.argv[i]+' to list');
    endpoints.push({
        url: process.argv[i],
        offlineSince: null,
        lastHeader: null,
        expectedClose: false
    })
}

let server = http.createServer(function () {});
server.listen(4000, function() {
    console.log('Server is listening on port :4000');
});

wsServer = new WebSocketServer({
    httpServer: server,
    autoAcceptConnections: true,
    keepalive: false
});

const selectEndpoint = () => {
    if (endpoints.length === 0) return null;
    if (endpoints.length === 1) return 0;
    for (let i = 0; i < endpoints.length; i++) {
        if (endpoints[i].offlineSince === null) return i;
    }
    let oldestCheck = 0;
    for (let i = 1; i < endpoints.length; i++) {
        if (endpoints[i].offlineSince < endpoints[oldestCheck].offlineSince) {
            oldestCheck = i;
        }
    }
    endpoints[oldestCheck].offlineSince = null;
    return oldestCheck;
};

const incomingConnection = (connection) => {
    console.log('Accepted incoming connection');

    let client = new WebSocketClient();
    let eth = null;
    let backlog = [];

    connection.on('message', function(message) {
        if (eth == null) {
            backlog.push(message.utf8Data);
            return;
        }
        sendData(eth, message.utf8Data);
    });

    let endpointId = selectEndpoint();
    if (endpointId == null) {
        console.log('No suitable endpoints available');
        close(connection)
    }

    let connected = false;

    client.on('connect', (ethConnection) => {
        connected = true;
        eth = ethConnection;
        outgoingConnection(connection, ethConnection, endpointId, backlog)
    });

    client.on('connectFailed', () => {
        console.log('Unable to connect to endpoint '+endpoints[endpointId].url);
        endpoints[endpointId].offlineSince = new Date();
        close(connection)
    });

    client.connect(endpoints[endpointId].url, null);

    connection.on('close', () => {
        endpoints[endpointId].expectedClose = true;
        close(eth);
        console.log('Disconnecting...')
    });

    setTimeout(() => {
        if (connected) return;
        console.log('Timed out trying to connect to endpoint '+endpoints[endpointId].url);
        endpoints[endpointId].offlineSince = new Date();
        close(connection);
        close(eth);
        client.abort();
    }, 5000)
};

const isBlockHeaderNotification = (data) => {
    let msg = JSON.parse(data);
    if (!('method' in msg)) return false;
    if (msg.method !== "eth_subscription") return false;
    if (!('result' in msg.params)) return false;
    return ('difficulty' in msg.params.result && 'parentHash' in msg.params.result);
};

const outgoingConnection = (connection, eth, endpointId, backlog) => {
    console.log('Connected to endpoint '+endpoints[endpointId].url);

    for (let i = 0; i < backlog.length; i++) {
        sendData(eth, backlog[i]);
    }

    let intervalId = setInterval(() => {
        if (new Date() - endpoints[endpointId].lastHeader > 180000) {
            endpoints[endpointId].offlineSince = new Date();
            endpoints[endpointId].expectedClose = true;
            console.log('Its been too long since we received a block header');
            close(connection);
            close(eth);
        }
    }, 60000);

    eth.on('close', () => {
        if (!endpoints[endpointId].expectedClose) {
            endpoints[endpointId].offlineSince = new Date();
            console.log('Lost connection to endpoint '+endpoints[endpointId].url+'!');
        }
        clearInterval(intervalId);
        close(connection)
    });

    eth.on('message', (message) => {
        sendData(connection, message.utf8Data);
        if (isBlockHeaderNotification(message.utf8Data)) {
            endpoints[endpointId].lastHeader = new Date()
        }
    });
};

wsServer.on('connect', incomingConnection);

const close = (connection) => {
    if (connection == null || !connection.connected) return;
    connection.close();
};

const sendData = (connection, data) => {
    if (connection == null || !connection.connected) return;
    connection.send(data);
};
