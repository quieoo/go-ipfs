package log

import (
	"encoding/json"
	"io"

	logging "github.com/ipfs/go-ipfs/Godeps/_workspace/src/github.com/whyrusleeping/go-logging"
)

// PoliteJSONFormatter marshals entries into JSON encoded slices (without
// overwriting user-provided keys). How polite of it!
type PoliteJSONFormatter struct{}

func (f *PoliteJSONFormatter) Format(calldepth int, r *logging.Record, w io.Writer) error {
	entry := make(map[string]interface{})
	entry["id"] = r.Id
	entry["level"] = r.Level
	entry["time"] = r.Time
	entry["module"] = r.Module
	entry["message"] = r.Message()
	err := json.NewEncoder(w).Encode(entry)
	if err != nil {
		return err
	}

	w.Write([]byte{'\n'})
	return nil
}
