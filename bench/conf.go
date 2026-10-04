package bench

import "math/rand/v2"

const (
	windowSize = int64(60)
	seqLen     = 1 << 20
)

func keySequence(nKeys int, zipf bool) []int32 {
	r := rand.New(rand.NewPCG(1, 2))
	seq := make([]int32, seqLen)

	var z *rand.Zipf
	if zipf {
		z = rand.NewZipf(r, 2, 1, uint64(nKeys-1))
	}
	for i := range seq {
		if zipf {
			seq[i] = int32(z.Uint64())
		} else {
			seq[i] = int32(r.IntN(nKeys))
		}
	}
	return seq
}
