# Chainlink ETH failover proxy

**NOTE:** This is currently a beta release, and should only be using on test networks. A stable release will be tagged
once it goes through sufficient testing.

## What does it do?

Instead of relying on a single connection between your Chainlink node and an ETH node (your own or EaaS), you can use
this proxy to fail over between multiple ETH nodes. All endpoints are just URLs, so there's no need for the nodes to be
on the local network.

### Features

* Automatic failover for any number of ETH nodes
* Connection timeouts
* Automatically checks if it's receiving block header notifications

## How to use

### Install

To start off, you need NPM/Node.js installed (and git, for cloning the repo).

Clone repository and enter directory:

```bash
git clone https://github.com/OracleFinder/ChainlinkEthFailover.git && cd ChainlinkEthFailover/
```

Install dependencies:

```bash
npm install
```

Make sure we set the correct permissions:

```bash
chmod +x ./index.js
```

### Start

This proxy will add any arguments to the script as ETH node endpoints.

```
node ./index.js [node-1] [node-2] [...] [node-n]
```

Example start command:

```bash
node ./index.js wss://cl-ropsten.fiews.io/v1/myApiKey ws://localhost:8546/
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
