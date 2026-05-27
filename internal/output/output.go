// Package output renders any slice/struct as table, JSON, or YAML.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"gopkg.in/yaml.v3"
)

type Format int

const (
	FormatTable Format = iota
	FormatJSON
	FormatYAML
)

func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "", "table":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return 0, fmt.Errorf("unknown output format %q (use table|json|yaml)", s)
	}
}

type Column struct {
	Header string
	Field  string // struct field name (top-level)
}

// Render writes rows in the chosen format. `rows` must be a slice or a single struct.
func Render(w io.Writer, rows any, cols []Column, f Format) error {
	switch f {
	case FormatJSON:
		return writeJSON(w, rows)
	case FormatYAML:
		return writeYAML(w, rows)
	default:
		return writeTable(w, rows, cols)
	}
}

func writeJSON(w io.Writer, rows any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func writeYAML(w io.Writer, rows any) error {
	b, err := yaml.Marshal(rows)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	return err
}

func writeTable(w io.Writer, rows any, cols []Column) error {
	v := reflect.ValueOf(rows)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Slice {
		// single struct → wrap in slice for uniform handling
		slice := reflect.MakeSlice(reflect.SliceOf(v.Type()), 1, 1)
		slice.Index(0).Set(v)
		v = slice
	}

	t := table.NewWriter()
	t.SetOutputMirror(w)

	headers := make(table.Row, len(cols))
	for i, c := range cols {
		headers[i] = c.Header
	}
	t.AppendHeader(headers)

	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		if item.Kind() == reflect.Ptr {
			item = item.Elem()
		}
		row := make(table.Row, len(cols))
		for j, c := range cols {
			fv := item.FieldByName(c.Field)
			if !fv.IsValid() {
				row[j] = ""
				continue
			}
			row[j] = fv.Interface()
		}
		t.AppendRow(row)
	}

	t.Render()
	return nil
}
