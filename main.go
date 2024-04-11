package main

import "gjlim2485/bandwidthawarecaching/codecache"

func main() {
	//codecache.Encoding(3, "data1.txt", "data2.txt", "data3.txt")
	codecache.Decoding("data1.txt-data2.txt-data3.txt.bin", "data1.txt-data2.txt-data3.txt.json", "data1.txt")
}
