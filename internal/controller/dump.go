package controller

import (
	"encoding/json"

	"github.com/davecgh/go-spew/spew"
)

func dumpResource(i interface{}) string {
	toDump, err := json.Marshal(i)
	if err != nil {
		return spew.Sdump(i)
	}
	return string(toDump)
}
