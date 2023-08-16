# Usage

This document provides instructions on how to use the Golang version of the backend code for the Pebble Voting App.

## Prerequisites

- Make sure you have Golang installed on your machine.
- Clone the repository to your local machine.

## Setting Up

1. Navigate to the `pebble-core` directory in the cloned repository.
2. Make a copy of the `anoncred1-params.bin` file and place it inside both the `pebble-core/cmd/server` and `pebble-core/cmd/client` directories. This file is used to initialize the `anoncred1` instance.

## Starting the Client Backend

1. Navigate to the `pebble-core/cmd/client` directory.
2. Open the `main.go` file and change the `projectRoot` variable to point to the root of your project on your respective machine.
3. Run the following command to start the client backend:
   ```bash
   go run main.go
   ```
   This will start the client backend, listening on the `127.0.0.1:8080` endpoint. You can edit the code to listen on a different endpoint if needed.

## Starting the Server Backend

1. Navigate to the `pebble-core/cmd/server` directory.
2. Run the following command to start the server backend, replacing the `ENDPOINT` variable with the desired endpoint:
   ```bash
   go run main.go mock ENDPOINT
   ```
   For example, a complete command could look like:
   ```bash
   go run main.go mock 127.0.0.1:8090
   ```

## Using the Application

Once both the client and server backends are running, you can use the Pebble Voting App ([refer to the README.md of the frontend repository](https://github.com/b4ba/pebble-frontend/tree/main) to register organizations, create elections,
