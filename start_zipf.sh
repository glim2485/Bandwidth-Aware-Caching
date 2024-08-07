enableMulticastValues=("true" "false")
zipfValues=("true")
seedValues=(7 89 506 777)
userCounts=(100)
userIterations=(50)
userCacheSizes=(50 25)
# file size in mb
dataSizes=(1000)
# max bandwidth in gbps
maxBandwidths=(5)
maxFilesValues=(200)
# edgeCacheSizeValues/maxFilesValues results in edge hit rate
edgeCacheSizeValues=(140 160 180 200)
maxCodedValues=(4)

# Initialize the iteration counter
iteration=1

# Loop through each combination of enableMulticast, seedValue, and other parameters
for maxFiles in "${maxFilesValues[@]}"; do
    for edgeCacheSize in "${edgeCacheSizeValues[@]}"; do
        for enableMulticast in "${enableMulticastValues[@]}"; do
            for seedValue in "${seedValues[@]}"; do
                for userCount in "${userCounts[@]}"; do
                    for userIteration in "${userIterations[@]}"; do
                        for userCacheSize in "${userCacheSizes[@]}"; do
                            for dataSize in "${dataSizes[@]}"; do
                                # Convert dataSize to bytes
                                dataSizeBytes=$((dataSize * 1000 * 1000))
                                for maxBandwidth in "${maxBandwidths[@]}"; do
                                    # Convert maxBandwidth to bits per second
                                    maxBandwidthBps=$((maxBandwidth * 1000 * 1000 * 1000))
                                    # Determine if enableCodecast should be true or false
                                    if [[ "$enableMulticast" == "true" ]]; then
                                        enableCodecastValues=("true" "false")
                                    else
                                        enableCodecastValues=("false")
                                    fi

                                    # Loop through each enableCodecast value
                                    for enableCodecast in "${enableCodecastValues[@]}"; do
                                    for zipfValue in "${zipfValues[@]}"; do
                                    for maxCoded in "${maxCodedValues[@]}"; do
                                        log_file="zipf_output${iteration}.log"
                                        
                                        go run . \
                                            -enableMulticast="$enableMulticast" \
                                            -enableCodecast="$enableCodecast" \
                                            -seedValue="$seedValue" \
                                            -userCount="$userCount" \
                                            -userIterations="$userIteration" \
                                            -userCacheSize="$userCacheSize" \
                                            -dataSize="$dataSizeBytes" \
                                            -maxBandwidth="$maxBandwidthBps" \
                                            -edgeCacheSize="$edgeCacheSize" \
                                            -maxFiles="$maxFiles" \
                                            -useZipf="$zipfValue" \
                                            -maxCodedItems="$maxCoded" \
                                            -saveFileName=dataLog_zipf_new.xlsx \
                                            2>&1 | tee "$log_file"
                                        
                                        iteration=$((iteration + 1))

                                        sleep 10
                                        done
                                        done
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
