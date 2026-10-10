package snapshot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"

	"github.com/cottondesu/stateripple/internal/graph"
)

func Read(reader io.Reader) (Snapshot, error) {
	b, err := io.ReadAll(io.LimitReader(reader, MaxInputBytes+1))
	if err != nil {
		return Snapshot{}, fmt.Errorf("read snapshot: %w", err)
	}
	if len(b) > MaxInputBytes {
		return Snapshot{}, fmt.Errorf("input exceeds %d bytes", MaxInputBytes)
	}
	if !utf8.Valid(b) || !validEscapedUnicode(b) {
		return Snapshot{}, fmt.Errorf("snapshot contains invalid Unicode")
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.UseNumber()
	if err = checkValue(d, 0, 0); err != nil {
		return Snapshot{}, err
	}
	if _, err = d.Token(); err != io.EOF {
		return Snapshot{}, fmt.Errorf("snapshot must contain exactly one JSON value")
	}
	var raw map[string]json.RawMessage
	if err = json.Unmarshal(b, &raw); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot object: %w", err)
	}
	rootFields := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		if key != "max_generations" {
			rootFields[key] = value
		}
	}
	if err = required(rootFields, "schema_version", "ruleset", "nodes", "edges", "seeds"); err != nil {
		return Snapshot{}, err
	}
	for _, group := range []struct {
		field  string
		fields []string
	}{
		{"nodes", []string{"id", "kind"}},
		{"edges", []string{"from", "to", "kind", "attenuation"}},
		{"seeds", []string{"id", "risk", "reason"}},
	} {
		field, fields := group.field, group.fields
		var objects []map[string]json.RawMessage
		if err = json.Unmarshal(raw[field], &objects); err != nil {
			return Snapshot{}, fmt.Errorf("%s must be an array of objects: %w", field, err)
		}
		for _, obj := range objects {
			if err = required(obj, fields...); err != nil {
				return Snapshot{}, fmt.Errorf("%s: %w", field, err)
			}
		}
	}
	s := Snapshot{MaxGenerations: DefaultGenerations}
	d = json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err = d.Decode(&s); err != nil {
		return Snapshot{}, fmt.Errorf("decode snapshot: %w", err)
	}
	return Normalize(s)
}

func required(object map[string]json.RawMessage, fields ...string) error {
	allowed := make(map[string]bool, len(fields))
	for _, f := range fields {
		allowed[f] = true
		if _, ok := object[f]; !ok {
			return fmt.Errorf("missing required field %q", f)
		}
	}
	keys := make([]string, 0, len(object))
	for key := range object {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		if !allowed[key] {
			return fmt.Errorf("unknown field %q", key)
		}
	}
	return nil
}

func checkValue(d *json.Decoder, depth, arrayLimit int) error {
	if depth > 16 {
		return fmt.Errorf("JSON nesting exceeds limit")
	}
	token, err := d.Token()
	if err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}
	if token == nil {
		return fmt.Errorf("null is not accepted in snapshot input")
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok {
				return fmt.Errorf("invalid object key")
			}
			if seen[name] {
				return fmt.Errorf("duplicate JSON field %q", name)
			}
			seen[name] = true
			limit := 0
			if depth == 0 {
				switch name {
				case "nodes":
					limit = graph.MaxNodes
				case "edges":
					limit = graph.MaxEdges
				case "seeds":
					limit = MaxSeeds
				}
			}
			if err = checkValue(d, depth+1, limit); err != nil {
				return err
			}
		}
	case '[':
		count := 0
		for d.More() {
			if arrayLimit > 0 && count >= arrayLimit {
				return fmt.Errorf("array exceeds element limit %d", arrayLimit)
			}
			count++
			if err = checkValue(d, depth+1, 0); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("unexpected JSON delimiter")
	}
	_, err = d.Token()
	return err
}
