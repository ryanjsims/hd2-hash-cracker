package stdlib

import (
	"bytes"
	_ "embed"
	"encoding/gob"

	"github.com/xypwn/hd2-hash-cracker/pattern"
)

//go:embed std_vars.gob
var stdVarsGob []byte

var Vars map[string]pattern.IrSegment

//go:embed target_murmur64a.txt
var TargetHashesMurmur64a []byte

//go:embed target_murmur64a_thin.txt
var TargetHashesMurmur64aThin []byte

//go:embed target_datalib.txt
var TargetHashesDatalib []byte

func init() {
	dec := gob.NewDecoder(bytes.NewReader(stdVarsGob))
	if err := dec.Decode(&Vars); err != nil {
		panic(err)
	}
}
