package flightradar

import (
	"encoding/csv"
	"io"
	"reflect"
	"strconv"
)

// WriteCSV writes a slice of structs to CSV with header inferred from `csv` tags or field names.
func WriteCSV(w io.Writer, slice any) error {
	rv := reflect.ValueOf(slice)
	if rv.Kind() != reflect.Slice {
		return nil
	}
	cw := csv.NewWriter(w)
	defer cw.Flush()

	// Header from first element
	if rv.Len() == 0 {
		return nil
	}
	t := rv.Index(0).Type()
	headers := make([]string, 0, t.NumField())
	fields := make([]int, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		} // unexported
		tag := f.Tag.Get("csv")
		name := tag
		if name == "" || name == "-" {
			name = f.Name
		}
		if name == "-" {
			continue
		}
		headers = append(headers, name)
		fields = append(fields, i)
	}
	if err := cw.Write(headers); err != nil {
		return err
	}

	for i := 0; i < rv.Len(); i++ {
		rowv := rv.Index(i)
		rec := make([]string, 0, len(fields))
		for _, idx := range fields {
			fv := rowv.Field(idx)
			rec = append(rec, toString(fv))
		}
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	return cw.Error()
}

func toString(v reflect.Value) string {
	if !v.IsValid() {
		return ""
	}
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return ""
		}
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		if v.Bool() {
			return "true"
		}
		return "false"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return itoa(v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return uitoa(v.Uint())
	case reflect.Float32, reflect.Float64:
		return ftoa(v.Float())
	default:
		return ""
	}
}

func itoa(n int64) string   { return strconv.FormatInt(n, 10) }
func uitoa(n uint64) string { return strconv.FormatUint(n, 10) }
func ftoa(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }
