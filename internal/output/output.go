// Package output renders results without selecting or changing generation inputs.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"scramble/internal/app"
	"scramble/internal/puzzle"
)

func JSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
func Text(w io.Writer, result app.Result) error {
	for _, s := range result.Scrambles {
		if _, err := fmt.Fprintln(w, s.Sequence); err != nil {
			return err
		}
	}
	return nil
}
func Puzzles(w io.Writer, descriptors []puzzle.Descriptor) error {
	table := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "ID\tNAME\tMETHODS"); err != nil {
		return err
	}
	for _, d := range descriptors {
		methods := make([]string, len(d.Methods))
		for i, m := range d.Methods {
			methods[i] = m.ID
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\n", d.ID, d.Name, strings.Join(methods, ", ")); err != nil {
			return err
		}
	}
	return table.Flush()
}
