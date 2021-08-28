# Chainlink ETH failover proxy

## What does it do?

Instead of relying on a single connection between your Chainlink node and an ETH node (your own or EaaS), you can use
this proxy to fail over between multiple ETH nodes. All endpoints are just URLs, so there's no need for the nodes to be
on the local network.

### Features

* Automatic failover for any number of ETH nodes
* Connection timeouts
* Automatically checks if it's receiving block header notifications
* Custom headers timeout through HEADERS_TIMEOUT environment variable (seconds)

## How to use

### Install (Docker)

Pull the image from Docker:

```bash
docker pull fiews/cl-eth-failover
```

Start the container: (Read more in Usage to learn how to set multiple endpoints)

```bash
docker run fiews/cl-eth-failover wss://cl-ropsten.fiews.io/v1/yourApiKey
```

### Install (manually)

To start off, you need Yarn/Node.js installed (and git, for cloning the repo).

Clone repository and enter directory:

```bash
git clone https://github.com/OracleFinder/ChainlinkEthFailover.git && cd ChainlinkEthFailover/
```

Install dependencies:

```bash
yarn install
```

Make sure we set the correct permissions:

```bash
chmod +x ./index.js
```

### Usage

This proxy will add any arguments to the script as ETH node endpoints.

```
docker run [-e HEADERS_TIMEOUT={}] fiews/cl-eth-failover [node-1] [node-2] [...] [node-n]
```

*If not using Docker, replace `docker run fiews/cl-eth-failover` with `node ./index.js`*

Example start command:

```bash
docker run fiews/cl-eth-failover wss://cl-ropsten.fiews.io/v1/myApiKey ws://localhost:8546/
```

This will output the following:

```
Adding endpoint wss://cl-ropsten.fiews.io/v1/myApiKey to list
Adding endpoint ws://localhost:8546/ to list
Server is listening on port :4000
```

### Chainlink configuration

To use this proxy with your Chainlink node, just set the `ETH_URL` environment variable to port 4000 of the instance
that's running the proxy. Eg. `ws://localhost:4000/`
