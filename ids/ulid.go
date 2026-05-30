// Package ids generates lexicographically-sortable identifiers (ULID).
// ULID = 48-bit millisecond timestamp + 80-bit randomness, Crockford base32.
package ids

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"
)

const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

var mu sync.Mutex

// New returns a new ULID string (26 chars, Crockford base32).
func New() string {
	ms := uint64(time.Now().UTC().UnixMilli())

	var b [16]byte
	// Bytes 0..5 hold the 48-bit timestamp (big endian).
	binary.BigEndian.PutUint64(b[0:8], ms<<16)

	// Bytes 6..15 hold 80 bits of entropy.
	mu.Lock()
	_, _ = rand.Read(b[6:])
	mu.Unlock()

	return encode(b)
}

// encode renders the 128-bit value as 26 Crockford base32 characters,
// following the canonical ULID layout (10 chars time, 16 chars entropy).
func encode(id [16]byte) string {
	e := crockford
	d := make([]byte, 26)
	d[0] = e[(id[0]&224)>>5]
	d[1] = e[id[0]&31]
	d[2] = e[(id[1]&248)>>3]
	d[3] = e[((id[1]&7)<<2)|((id[2]&192)>>6)]
	d[4] = e[(id[2]&62)>>1]
	d[5] = e[((id[2]&1)<<4)|((id[3]&240)>>4)]
	d[6] = e[((id[3]&15)<<1)|((id[4]&128)>>7)]
	d[7] = e[(id[4]&124)>>2]
	d[8] = e[((id[4]&3)<<3)|((id[5]&224)>>5)]
	d[9] = e[id[5]&31]
	d[10] = e[(id[6]&248)>>3]
	d[11] = e[((id[6]&7)<<2)|((id[7]&192)>>6)]
	d[12] = e[(id[7]&62)>>1]
	d[13] = e[((id[7]&1)<<4)|((id[8]&240)>>4)]
	d[14] = e[((id[8]&15)<<1)|((id[9]&128)>>7)]
	d[15] = e[(id[9]&124)>>2]
	d[16] = e[((id[9]&3)<<3)|((id[10]&224)>>5)]
	d[17] = e[id[10]&31]
	d[18] = e[(id[11]&248)>>3]
	d[19] = e[((id[11]&7)<<2)|((id[12]&192)>>6)]
	d[20] = e[(id[12]&62)>>1]
	d[21] = e[((id[12]&1)<<4)|((id[13]&240)>>4)]
	d[22] = e[((id[13]&15)<<1)|((id[14]&128)>>7)]
	d[23] = e[(id[14]&124)>>2]
	d[24] = e[((id[14]&3)<<3)|((id[15]&224)>>5)]
	d[25] = e[id[15]&31]
	return string(d)
}

// Prefixed returns a kind-prefixed identifier, e.g. "proj_01J...".
func Prefixed(prefix string) string {
	return prefix + "_" + New()
}
