package pretty

import (
	"encoding/json"
	"fmt"
)

// Object ...
func Object(o interface{}) string {
	b, err := json.MarshalIndent(o, "", "\t")
	if err != nil {
		return fmt.Sprint(o)
	}
	return string(b)
}
