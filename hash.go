package gcache

import (
	"hash/crc64"
	"hash/fnv"
)

//HashCalculator interface about hash funcs
type HashCalculator func(string) uint64

func calcHashFNV(str string) uint64 {
	hash := fnv.New64a()
	hash.Write([]byte(str))
	return hash.Sum64()
}

func calcHashCRC(str string) uint64 {
	crc64Table := crc64.MakeTable(0xC96C5795D7870F42)
	hash := crc64.New(crc64Table)
	hash.Write([]byte(str))
	return hash.Sum64()
}

func djb33(str string) uint64 {
	var (
		l = uint64(len(str))
		d = 2189 + l
		i = uint64(0)
	)

	if l >= 4 {
		for i < l-4 {
			d = (d * 33) ^ uint64(str[i])
			d = (d * 33) ^ uint64(str[i+1])
			d = (d * 33) ^ uint64(str[i+2])
			d = (d * 33) ^ uint64(str[i+3])
			i += 4
		}
	}
	switch l - i {
	case 1:
	case 2:
		d = (d * 33) ^ uint64(str[i])
	case 3:
		d = (d * 33) ^ uint64(str[i])
		d = (d * 33) ^ uint64(str[i+1])
	case 4:
		d = (d * 33) ^ uint64(str[i])
		d = (d * 33) ^ uint64(str[i+1])
		d = (d * 33) ^ uint64(str[i+2])
	}
	return d ^ (d >> 16)
}

func calcSUM(str string) uint64 {
	var (
		l = len(str)
		r = uint64(l)
		i = 0
	)

	if l >= 4 {
		for i < l-4 {
			r += uint64(str[i]) >> 2
			r += uint64(str[i+1]) >> 2
			r += uint64(str[i+2]) >> 2
			r += uint64(str[i+3]) >> 2
			i += 4
		}
	}
	switch l - i {
	case 1:
	case 2:
		r += uint64(str[i]) >> 2
	case 3:
		r += uint64(str[i]) >> 2
		r += uint64(str[i+1]) >> 2
	case 4:
		r += uint64(str[i]) >> 2
		r += uint64(str[i+1]) >> 2
		r += uint64(str[i+2]) >> 2
	}
	return r ^ (r >> 16)
}
