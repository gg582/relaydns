package randpool

import (
	"crypto/rand"
	"io"
	"log"
	"sync"

	"golang.org/x/crypto/chacha20"
)

var _csprng_fallback = func() *chacha20.Cipher {
	var initdata [12 + 32]byte // 12 byte nonce, 32 byte key
	_, err := io.ReadFull(rand.Reader, initdata[:])
	if err != nil {
		panic(err)
	}
	c, err := chacha20.NewUnauthenticatedCipher(initdata[12:], initdata[:12])
	if err != nil {
		panic(err)
	}
	return c
}()

type chacha20rng struct {
	c    *chacha20.Cipher
	used uint64
}

var _chacha20rngPool sync.Pool = sync.Pool{
	New: func() interface{} {
		var initdata [12 + 32]byte // 12 byte nonce, 32 byte key
		_, err := rand.Read(initdata[:])
		if err != nil {
			// if system rand fails, use fallback and print log
			log.Println("randpool: chacha20rng init failed to read from system rand, using fallback")
			_csprng_fallback.XORKeyStream(initdata[:], initdata[:])
		}
		c, err := chacha20.NewUnauthenticatedCipher(initdata[12:], initdata[:12])
		if err != nil {
			panic(err) // should never happen
		}
		return &chacha20rng{
			c: c,
		}
	},
}

func _chacha20rng() *chacha20rng {
	return _chacha20rngPool.Get().(*chacha20rng)
}

func _CHACHA20_RAND(dst []byte) {
	c := _chacha20rng()
	c.used += uint64(len(dst))
	c.c.XORKeyStream(dst, dst)
	if c.used < 50*1<<30 {
		// Return to pool only if we haven't used more than 50GiB
		_chacha20rngPool.Put(c)
	}
}

func CSPRNG_RAND(dst []byte) {
	_CHACHA20_RAND(dst)
}
