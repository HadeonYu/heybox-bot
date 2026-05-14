package api

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	crand "crypto/rand"
)

const alphabet = "AB45STUVWZEFGJ6CH01D237IXYPQRKLMN89"

func vm(e int32) int32 {
	if (int32(128) & e) != 0 {
		return int32(255) & ((e << 1) ^ 27)
	}
	return e << 1
}

func qm(e int32) int32 {
	return vm(e) ^ e
}

func dm(e int32) int32 {
	return qm(vm(e))
}

func ym(e int32) int32 {
	return dm(qm(vm(e)))
}

func gm(e int32) int32 {
	return ym(e) ^ dm(e) ^ qm(e)
}

func km(e []int32) []int32 {
	r := make([]int32, len(e))
	copy(r, e)

	t := []int32{0, 0, 0, 0}
	t[0] = gm(r[0]) ^ ym(r[1]) ^ dm(r[2]) ^ qm(r[3])
	t[1] = qm(r[0]) ^ gm(r[1]) ^ ym(r[2]) ^ dm(r[3])
	t[2] = dm(r[0]) ^ qm(r[1]) ^ gm(r[2]) ^ ym(r[3])
	t[3] = ym(r[0]) ^ dm(r[1]) ^ qm(r[2]) ^ gm(r[3])

	r[0], r[1], r[2], r[3] = t[0], t[1], t[2], t[3]
	return r
}

func jsSlice0(s string, n int) string {
	end := n
	if n < 0 {
		end = len(s) + n
	}
	if end < 0 {
		end = 0
	}
	if end > len(s) {
		end = len(s)
	}
	return s[:end]
}

func av(s, table string, n int) string {
	base := jsSlice0(table, n)

	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteByte(base[int(s[i])%len(base)])
	}
	return b.String()
}

func sv(s, table string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		b.WriteByte(table[int(s[i])%len(table)])
	}
	return b.String()
}

func interleaveLimit20(parts ...string) string {
	maxLen := 0
	for _, p := range parts {
		if len(p) > maxLen {
			maxLen = len(p)
		}
	}

	var b strings.Builder
	for i := 0; i < maxLen; i++ {
		for _, p := range parts {
			if i < len(p) {
				b.WriteByte(p[i])
			}
		}
	}

	out := b.String()
	if len(out) > 20 {
		return out[:20]
	}
	return out
}

func normalizePath(path string) string {
	parts := strings.FieldsFunc(path, func(r rune) bool {
		return r == '/'
	})
	return "/" + strings.Join(parts, "/") + "/"
}

func md5Lower(s string) string {
	sum := md5.Sum([]byte(s))
	return fmt.Sprintf("%x", sum)
}

func md5Upper(s string) string {
	sum := md5.Sum([]byte(s))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

func makeNonce(ts int64) string {
	// 浏览器里是 MD5(ts + Math.random().toString()).toUpperCase()
	// 服务端主要校验 nonce 和 hkey 是否匹配，所以这里用 crypto/rand 生成随机源也可以。
	buf := make([]byte, 16)
	if _, err := crand.Read(buf); err != nil {
		panic(err)
	}

	raw := fmt.Sprintf("%d%s", ts, hex.EncodeToString(buf))
	return md5Upper(raw)
}

func calcHKey(path string, ts int64, nonce string) string {
	// JS 里 lv.g(e, t, r) => ov(e, t + 1, r)
	signTime := ts + 1

	e := normalizePath(path)

	mix := interleaveLimit20(
		av(fmt.Sprintf("%d", signTime), alphabet, -2),
		sv(e, alphabet),
		sv(nonce, alphabet),
	)

	h := md5Lower(mix)

	last6 := h[len(h)-6:]
	arr := make([]int32, 0, 6)
	for i := 0; i < len(last6); i++ {
		arr = append(arr, int32(last6[i]))
	}

	var total int32
	for _, v := range km(arr) {
		total += v
	}

	suffix := fmt.Sprintf("%d", total%100)
	if len(suffix) < 2 {
		suffix = "0" + suffix
	}

	prefix := av(h[:5], alphabet, -4)

	return prefix + suffix
}

func makeHeyboxSign(path string) (hkey string, ts int64, nonce string) {
	ts = time.Now().Unix()
	nonce = makeNonce(ts)
	hkey = calcHKey(path, ts, nonce)
	return
}
