package data

import (
	"fmt"
	"strconv"
)

type Runtime int32

// Method isn't receiving *Runtime because:
// Runtime isn't a struct it's not memory-heavy, also this method will work on both value and pointer
func (r Runtime) MarshalJSON() ([]byte, error) {
	jsonValue := fmt.Sprintf("%d mins", r)
	quoteWrapped := strconv.Quote(jsonValue)

	return []byte(quoteWrapped), nil
}
