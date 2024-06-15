#!/bin/bash

enableMulticastValues=("true" "false")
seedValues=("7")
userCounts=(100)
userIterations=(50)
userCacheSizes=(25)
dataSizes=(500)
maxBandwidths=(2500)
edgeCacheSizeMultiplierValues=(10)
maxFilesValues=(50)

# Loop through each combination of enableMulticast, seedValue, and other parameters
for maxFiles in "${maxFilesValues[@]}"; do
    for edgeCacheSizeMultiplier in "${edgeCacheSizeMultiplierValues[@]}"; do
        for enableMulticast in "${enableMulticastValues[@]}"; do
            for seedValue in "${seedValues[@]}"; do
                for userCount in "${userCounts[@]}"; do
                    for userIteration in "${userIterations[@]}"; do
                        for userCacheSize in "${userCacheSizes[@]}"; do
                            for dataSize in "${dataSizes[@]}"; do
                                for maxBandwidth in "${maxBandwidths[@]}"; do
                                    # Determine if enableCodecast should be true or false
                                    if [[ "$enableMulticast" == "true" ]]; then
                                        enableCodecastValues=("false" "true")
                                    else
                                        enableCodecastValues=("false")
                                    fi

                                    # Loop through each enableCodecast value
                                    for enableCodecast in "${enableCodecastValues[@]}"; do
                                        go run . \
                                            -enableMulticast="$enableMulticast" \
                                            -enableCodecast="$enableCodecast" \
                                            -seedValue="$seedValue" \
                                            -userCount="$userCount" \
                                            -userIterations="$userIteration" \
                                            -userCacheSize="$userCacheSize" \
                                            -dataSize="$dataSize" \
                                            -maxBandwidth="$maxBandwidth" \
                                            -edgeCacheSizeMultiplier="$edgeCacheSizeMultiplier" \
                                            -maxFiles="$maxFiles"
                                        
                                        # Add any additional logic or commands after each run if needed
                                    done
                                done
                            done
                        done
                    done
                done
            done
        done
    done
done
