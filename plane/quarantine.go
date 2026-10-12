package plane

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"sort"
)

// QuarantineEvent is the categorical, peer-attested record emitted when a
// fundamental cycle is frustrated. It is violation-channel evidence: it
// localizes to specific planes, and a receiving side must verify the signature
// and ignore unsigned events.
type QuarantineEvent struct {
	Cycle     [2]int  `json:"cycle"`
	Planes    []int   `json:"planes"`
	Phi       float64 `json:"phi"`
	Action    string  `json:"action"`
	Signature string  `json:"signature"`
}

// QuarantineOutcome is the result of the closing move: the rebuilt tree, the
// signed events, and whether any quarantine happened.
type QuarantineOutcome struct {
	Tree        *Tree
	Events      []QuarantineEvent
	Quarantined bool
}

// DefaultQuarantineKey is the deterministic development signing key. Production
// callers should supply their own key.
func DefaultQuarantineKey() []byte {
	h := sha256.Sum256([]byte("isotope-plane-quarantine-v1"))
	return h[:]
}

// SignEvent returns a copy of e carrying an HMAC-SHA256 signature over its
// canonical bytes. The signature is deterministic, so signed output stays
// bit-identical across runs.
func SignEvent(key []byte, e QuarantineEvent) (QuarantineEvent, error) {
	msg := canonicalEvent(e)
	mac := hmac.New(sha256.New, key)
	if _, err := mac.Write(msg); err != nil {
		return e, err
	}
	e.Signature = base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return e, nil
}

// VerifyEvent reports whether e carries a valid signature for key. Unsigned or
// tampered events verify as false.
func VerifyEvent(key []byte, e QuarantineEvent) bool {
	sig := e.Signature
	e.Signature = ""
	want, err := SignEvent(key, e)
	if err != nil {
		return false
	}
	return hmac.Equal([]byte(sig), []byte(want.Signature))
}

// Quarantine runs the closing move: it finds the first frustrated fundamental
// cycle, attributes it to all three suspects (the two endpoints of the non-tree
// edge and their lowest common ancestor), drops the responsible tree edge,
// rebuilds the tree without it, re-caches transports, and repeats until no
// cycle is frustrated or no further edge can be dropped. It is deterministic.
func Quarantine(space Space, tensor *Tensor, z []complex128, tol float64, key []byte) (*QuarantineOutcome, error) {
	var forbidden []Edge
	var events []QuarantineEvent
	var last *Tree
	maxIter := len(space.Planes) + 1
	for iter := 0; iter <= maxIter; iter++ {
		tr, err := BuildTree(space, tensor, forbidden)
		if err != nil {
			return nil, err
		}
		tr.CacheTransports(z)
		last = tr
		cycles, phis, err := tr.FrustratedCycles(z, tol)
		if err != nil {
			return nil, err
		}
		if len(cycles) == 0 {
			break
		}
		e := cycles[0]
		lca := tr.LCA(e.J, e.K)
		planes := sortedUniqueInts(e.J, e.K, lca)
		ev := QuarantineEvent{Cycle: [2]int{e.J, e.K}, Planes: planes, Phi: phis[0], Action: "quarantine"}
		ev, err = SignEvent(key, ev)
		if err != nil {
			return nil, err
		}
		events = append(events, ev)

		resp := e.J
		if tr.Depth[e.K] > tr.Depth[e.J] || (tr.Depth[e.K] == tr.Depth[e.J] && e.K > e.J) {
			resp = e.K
		}
		p := tr.Parent[resp]
		if p < 0 {
			break
		}
		drop := Edge{J: resp, K: p}
		if drop.J > drop.K {
			drop.J, drop.K = drop.K, drop.J
		}
		if isForbidden(forbidden, drop.J, drop.K) {
			break
		}
		forbidden = append(forbidden, drop)
	}
	return &QuarantineOutcome{Tree: last, Events: events, Quarantined: len(events) > 0}, nil
}

func canonicalEvent(e QuarantineEvent) []byte {
	return []byte(fmt.Sprintf("plane-quarantine|v1|%d,%d|%v|%.17g|%s",
		e.Cycle[0], e.Cycle[1], e.Planes, e.Phi, e.Action))
}

func sortedUniqueInts(xs ...int) []int {
	cp := append([]int(nil), xs...)
	sort.Ints(cp)
	out := cp[:0]
	for i, v := range cp {
		if i == 0 || v != cp[i-1] {
			out = append(out, v)
		}
	}
	return out
}
