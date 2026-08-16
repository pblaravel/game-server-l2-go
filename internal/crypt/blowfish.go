package crypt

// Engine is the L2J BlowfishEngine: standard Blowfish key schedule with
// little-endian 32-bit block word packing (NOT RFC/OpenSSL big-endian).
type Engine struct {
	p  [18]uint32
	s0 [256]uint32
	s1 [256]uint32
	s2 [256]uint32
	s3 [256]uint32
}

func NewEngine(key []byte) *Engine {
	e := &Engine{}
	e.init(key)
	return e
}

func (e *Engine) init(key []byte) {
	e.s0 = blowfishS0
	e.s1 = blowfishS1
	e.s2 = blowfishS2
	e.s3 = blowfishS3
	e.p = blowfishP

	keyIndex := 0
	keyLen := len(key)
	for i := 0; i < 18; i++ {
		var data uint32
		for j := 0; j < 4; j++ {
			data = data<<8 | uint32(key[keyIndex]&0xFF)
			keyIndex++
			if keyIndex >= keyLen {
				keyIndex = 0
			}
		}
		e.p[i] ^= data
	}
	e.processTable(0, 0, e.p[:])
	e.processTable(e.p[16], e.p[17], e.s0[:])
	e.processTable(e.s0[254], e.s0[255], e.s1[:])
	e.processTable(e.s1[254], e.s1[255], e.s2[:])
	e.processTable(e.s2[254], e.s2[255], e.s3[:])
}

func (e *Engine) f(x uint32) uint32 {
	return ((e.s0[x>>24] + e.s1[x>>16&0xFF]) ^ e.s2[x>>8&0xFF]) + e.s3[x&0xFF]
}

func (e *Engine) processTable(xl, xr uint32, table []uint32) {
	for s := 0; s < len(table); s += 2 {
		xl ^= e.p[0]
		xr ^= e.f(xl) ^ e.p[1]
		xl ^= e.f(xr) ^ e.p[2]
		xr ^= e.f(xl) ^ e.p[3]
		xl ^= e.f(xr) ^ e.p[4]
		xr ^= e.f(xl) ^ e.p[5]
		xl ^= e.f(xr) ^ e.p[6]
		xr ^= e.f(xl) ^ e.p[7]
		xl ^= e.f(xr) ^ e.p[8]
		xr ^= e.f(xl) ^ e.p[9]
		xl ^= e.f(xr) ^ e.p[10]
		xr ^= e.f(xl) ^ e.p[11]
		xl ^= e.f(xr) ^ e.p[12]
		xr ^= e.f(xl) ^ e.p[13]
		xl ^= e.f(xr) ^ e.p[14]
		xr ^= e.f(xl) ^ e.p[15]
		xl ^= e.f(xr) ^ e.p[16]
		xr ^= e.p[17]
		table[s] = xr
		table[s+1] = xl
		xr = xl
		xl = table[s]
	}
}

func bytesTo32LE(src []byte, i int) uint32 {
	return uint32(src[i]) | uint32(src[i+1])<<8 | uint32(src[i+2])<<16 | uint32(src[i+3])<<24
}

func bits32ToBytesLE(in uint32, dst []byte, i int) {
	dst[i] = byte(in)
	dst[i+1] = byte(in >> 8)
	dst[i+2] = byte(in >> 16)
	dst[i+3] = byte(in >> 24)
}

func (e *Engine) EncryptBlock(src []byte, srcIndex int) {
	xl := bytesTo32LE(src, srcIndex)
	xr := bytesTo32LE(src, srcIndex+4)
	xl ^= e.p[0]
	xr ^= e.f(xl) ^ e.p[1]
	xl ^= e.f(xr) ^ e.p[2]
	xr ^= e.f(xl) ^ e.p[3]
	xl ^= e.f(xr) ^ e.p[4]
	xr ^= e.f(xl) ^ e.p[5]
	xl ^= e.f(xr) ^ e.p[6]
	xr ^= e.f(xl) ^ e.p[7]
	xl ^= e.f(xr) ^ e.p[8]
	xr ^= e.f(xl) ^ e.p[9]
	xl ^= e.f(xr) ^ e.p[10]
	xr ^= e.f(xl) ^ e.p[11]
	xl ^= e.f(xr) ^ e.p[12]
	xr ^= e.f(xl) ^ e.p[13]
	xl ^= e.f(xr) ^ e.p[14]
	xr ^= e.f(xl) ^ e.p[15]
	xl ^= e.f(xr) ^ e.p[16]
	xr ^= e.p[17]
	bits32ToBytesLE(xr, src, srcIndex)
	bits32ToBytesLE(xl, src, srcIndex+4)
}

func (e *Engine) DecryptBlock(src []byte, srcIndex int) {
	xl := bytesTo32LE(src, srcIndex)
	xr := bytesTo32LE(src, srcIndex+4)
	xl ^= e.p[17]
	xr ^= e.f(xl) ^ e.p[16]
	xl ^= e.f(xr) ^ e.p[15]
	xr ^= e.f(xl) ^ e.p[14]
	xl ^= e.f(xr) ^ e.p[13]
	xr ^= e.f(xl) ^ e.p[12]
	xl ^= e.f(xr) ^ e.p[11]
	xr ^= e.f(xl) ^ e.p[10]
	xl ^= e.f(xr) ^ e.p[9]
	xr ^= e.f(xl) ^ e.p[8]
	xl ^= e.f(xr) ^ e.p[7]
	xr ^= e.f(xl) ^ e.p[6]
	xl ^= e.f(xr) ^ e.p[5]
	xr ^= e.f(xl) ^ e.p[4]
	xl ^= e.f(xr) ^ e.p[3]
	xr ^= e.f(xl) ^ e.p[2]
	xl ^= e.f(xr) ^ e.p[1]
	xr ^= e.p[0]
	bits32ToBytesLE(xr, src, srcIndex)
	bits32ToBytesLE(xl, src, srcIndex+4)
}
