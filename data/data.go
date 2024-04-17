package data

import "math/rand"

func GetZipfDistribution(seed int64) *rand.Zipf {
	zipfSeed := rand.New(rand.NewSource(seed))
	zipf := rand.NewZipf(zipfSeed, 1.1, 1, 100)
	return zipf
}
