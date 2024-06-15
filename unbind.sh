#!/bin/bash

# Define the range of ports to unbind
START_PORT=50000
END_PORT=60001

# Function to unbind a port
unbind_port() {
    port=$1
    # Find the process using the port
    pid=$(lsof -ti :$port)
    if [ ! -z "$pid" ]; then
        echo "Killing process $pid on port $port"
        # Kill the process
        kill -9 $pid
    fi
}

export -f unbind_port

# Generate the list of ports and process them in parallel
seq $START_PORT $END_PORT | parallel -j 4 unbind_port

echo "All specified ports have been unbound."