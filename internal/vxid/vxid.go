// Package vxid supports translating between internal and external varangian ids
package vxid

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/lithammer/shortuuid/v3"
)

const (
	pfxDelim string = "_"
)

type pfxMap struct {
	Organization string
	User         string
	Account      string
	Portfolio    string
	Strategy     string
	Instrument   string
	Transaction  string
	Lot          string
}

var (
	// PfxMap human-readable friendly mapping for resources to their vxid prefixes
	// TODO: load this into memory from a database
	PfxMap = pfxMap{
		Organization: "org",
		User:         "usr",
		Account:      "acct",
		Portfolio:    "prt",
		Strategy:     "str",
		Instrument:   "inst",
		Transaction:  "txn",
		Lot:          "lot"}
)

// Encode converts a internal id (vid) to an external id (vxid)
func Encode(vid string, pfx string) (vxid string, err error) {
	if vid == "" {
		return
	}

	if pfx != "" {
		pfx = pfx + pfxDelim
	}

	vuuid, err := uuid.Parse(vid)
	if err != nil {
		return
	}
	vxid = pfx + shortuuid.Encoder.Encode(shortuuid.DefaultEncoder, vuuid)

	return
}

// Encodes converts a slice of internal ids (vids) into a slice of external ids (vxids)
func Encodes(vids []string, pfxs []string) ([]string, error) {
	var vxids []string
	var pfx string

	if vids[0] == "" {
		return vids, nil
	}

	// if passing more than one prefix, make sure slice size matches vids slice size
	if len(pfxs) > 1 && (len(vids) != len(pfxs)) {
		return nil, fmt.Errorf("vids and pfxs slice lengths are different sizes %d, %d", len(vids), len(pfxs))
	}

	pfx = pfxs[0]
	for i := 0; i < len(vids); i++ {
		if len(pfxs) > 1 {
			pfx = pfxs[i]
		}
		vxid, err := Encode(vids[i], pfx)
		if err != nil {
			return nil, err
		}
		vxids = append(vxids, vxid)
	}

	return vxids, nil
}

// Decode converts an external id (vxid) to an internal (vid)
func Decode(vxid string) (vid string, err error) {
	if vxid == "" {
		return
	}

	vxidComponents := strings.Split(string(vxid), "_")

	vxidIndex := 0
	if len(vxidComponents) > 1 {
		vxidIndex = 1
	}

	vuuid, err := shortuuid.Encoder.Decode(shortuuid.DefaultEncoder, vxidComponents[vxidIndex])
	if err != nil {
		return
	}
	vid = vuuid.String()

	return
}

// Decodes converts a slice of external ids (vxids) to a slice of internal ids (vids)
func Decodes(vxids []string) ([]string, error) {
	var vids []string

	if vxids[0] == "" {
		return vxids, nil
	}

	for _, vxid := range vxids {
		vid, err := Decode(vxid)
		if err != nil {
			return nil, err
		}
		vids = append(vids, vid)
	}

	return vids, nil
}
