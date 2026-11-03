package semantic

import (
	"encoding/json"
	"io"
)

func jsonNewEncoder(w io.Writer, v any) error {
	return json.NewEncoder(w).Encode(v)
}
