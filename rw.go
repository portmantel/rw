package rw

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"log"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
)

// FileExists checks if a file exists at the given filepath.
// Works on symbolic links as well as real files.
func FileExists(fp string) bool {
	_, err := os.Stat(fp)
	if err != nil {
		// check the error message against system built-in
		// could error due to access issues
		log.Println(err)
		return !os.IsNotExist(err)
	}
	// no error means file exists
	return true
}

// ValidateFilepath returns the absolute path of the
// file if it exists, or an empty string.
func ValidateFilepath(file string) string {
	fp, err := filepath.Abs(file)
	if err != nil {
		log.Println(err)
		return ""
	}
	_, err = os.Stat(fp)
	if err != nil {
		log.Println(err)
		return ""
	}
	return fp
}

func ReadCsvFile(fn string) (raw [][]string) {
	file, err := os.Open(ValidateFilepath(fn))
	if err != nil {
		log.Printf("opening '%s' - %s\n", fn, err)
		return
	}
	r := csv.NewReader(file)
	for {
		row, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("reading - %s\n", err)
			return
		}
		raw = append(raw, row)
	}
	return
}

// NewCsvFile creates a CSV file and returns a Writer object.
// Returning a writer means the file must be closed externally.
func NewCsvFile(fn string) *csv.Writer {
	fp, err := filepath.Abs(fn)
	if err != nil {
		log.Printf("can't abs path to '%s' - %s\n", fn, err)
		return nil
	}
	if FileExists(fp) {
		log.Printf("file already exists at '%s'\n", fp)
		return nil
	}
	newFile, err := os.Create(fp)
	if err != nil {
		log.Printf("creating '%s' - %s\n", fp, err)
		return nil
	}
	return csv.NewWriter(newFile)
}

func CommaSep(fn string, headers []string, values [][]string) {
	// create the writer
	cw := NewCsvFile(fn)
	if cw == nil {
		log.Printf("failed to write new csv\n")
		return
	}
	// first row is separated and defines the number of columns
	err := cw.Write(headers)
	if err != nil {
		log.Printf("failing writing csv headers - %s\n", err)
		return
	}
	// write the rest of the data to follow
	for i, v := range values {
		err = cw.Write(v)
		if err != nil {
			log.Printf("writing row '%d' to csv - %s\n", i, err)
		}
	}
	// write to file
	cw.Flush()
	log.Printf("wrote '%d' lines to '%s'\n", len(values)+1, fn)
}

// LoadLines reads a file into memory and returns a slice of its lines
func LoadLines(fn string) (res []string) {
	f, err := os.Open(ValidateFilepath(fn))
	if err != nil {
		log.Printf("opening '%s' - %s\n", fn, err)
		return res
	}
	defer f.Close()
	s := bufio.NewScanner(f)
	for s.Scan() {
		res = append(res, strings.TrimSpace(s.Text()))
	}
	return
}

// LineInFileContains is a poor-mans grep, returning a string index
// of which line contained the match and the complete line.
func LineInFileContains(fn, search string) (i []int, lines []string) {
	allLines := LoadLines(fn)
	for n, l := range allLines {
		if strings.Contains(l, search) {
			i = append(i, n)
			lines = append(lines, l)
		}
	}
	if len(lines) == 0 {
		log.Println("no matches")
	}
	return
}

// ReadFileBytes opens the file and returns the entire content
// as raw bytes, nil if an error occurs.
func ReadFileBytes(fn string) []byte {
	f, err := os.ReadFile(ValidateFilepath(fn))
	if err != nil {
		log.Println(err)
		return nil
	}
	return f
}

// Returns a string of properly indented JSON,
// most useful for stringifying structs
func JsonPretty(v interface{}) string {
	pretty, err := json.MarshalIndent(v, "", "    ")
	if err != nil {
		return err.Error()
	}
	return string(pretty)
}

// Returns a string of JSON without indents and line-breaks.
func JsonFlat(v interface{}) string {
	flat, err := json.Marshal(v)
	if err != nil {
		return err.Error()
	}
	return string(flat)
}

// Returns a string of indented XML, or a string error
func XmlPretty(v interface{}) string {
	raw, err := formatXML([]byte(v.(string)))
	if err != nil {
		return err.Error()
	} else {
		return string(raw)
	}
}

// indents the raw XML
func formatXML(data []byte) ([]byte, error) {
	b := &bytes.Buffer{}
	decoder := xml.NewDecoder(bytes.NewReader(data))
	encoder := xml.NewEncoder(b)
	encoder.Indent("", "    ")
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			encoder.Flush()
			return b.Bytes(), nil
		}
		if err != nil {
			return nil, err
		}
		err = encoder.EncodeToken(token)
		if err != nil {
			return nil, err
		}
	}
}

func ExistsInList(query string, list []string) bool {
	for _, l := range list {
		if l == query {
			return true
		}
	}
	return false
}

func AppendIfUnique(sl []string, val string) []string {
	for _, v := range sl {
		if v == val {
			return sl
		}
	}
	sl = append(sl, val)
	return sl
}

// ConcatListNicely returns a single string where all slice
// values are separated with ', ' and the space and comma are
// removed from the last entry, for printing.
func ConcatListNicely(list []string) string {
	var res string
	for _, r := range list {
		if r != "" {
			res += fmt.Sprintf("%s, ", r)
		}

	}
	return strings.TrimRight(res, ", ")
}

// basic wrapper around strings.Split useful on multi-line queries
func SplitLines(lines []string, sep string) (splines [][]string) {
	for _, l := range lines {
		tmp := strings.Split(l, sep)
		splines = append(splines, tmp)
	}
	return
}

// Stops reading after 2 consecutive empty lines.
func ReadLinesFromStdin() (lines []string) {
	var skip bool
	reader := bufio.NewReaderSize(os.Stdin, 1024*1024)
	for {
		a, _, err := reader.ReadLine()
		if err == io.EOF {
			if skip {
				break
			} else {
				skip = true
				continue
			}
		} else if err != nil {
			log.Printf("reading - %s\n", err)
			return
		}
		line := strings.TrimRight(string(a), "\r\n")
		if line == "" {
			if skip {
				break
			} else {
				skip = true
				continue
			}
		}
		skip = false
		// great place to format/parse input
		lines = append(lines, line)
	}
	return
}

// ReadFromStdin returns a single line from standard in, stripping the newline.
func ReadFromStdin() string {
	reader := bufio.NewReaderSize(os.Stdin, 1024*1024)
	a, _, err := reader.ReadLine()
	if err == io.EOF {
		return ""
	} else if err != nil {
		panic(err)
	}
	return strings.TrimRight(string(a), "\r\n")
}

// fs fills the space with # of "-" to match length of header string above
func fs(p string) (f string) {
	f = strings.Repeat("-", len(p))
	return f
}

// TabFlex is a flexible, generic way to display tab-delimited text.
// Arguments are the headers, which will determine the number of columns,
// and a set of values, shown simply as a slice of unknown elements.
func TabFlex(headers []string, values [][]interface{}) {
	// create the writer
	tw := new(tabwriter.Writer).Init(os.Stdout, 0, 8, 2, ' ', 0)
	// define the columns
	var count int
	for _, h := range headers {
		fmt.Fprintf(tw, "%v\t", h)
		count++
	}
	fmt.Fprintf(tw, "\n")
	// stylistically write '-' grid matching the header width
	for _, h := range headers {
		fmt.Fprintf(tw, "%v\t", fs(h))
	}
	fmt.Fprintf(tw, "\n")
	// iterate over each slice, printing each element of the slice
	// empty strings "" will be used for less values than headers
	for _, v := range values {
		for i := 0; i < len(headers); i++ {
			if len(v) <= i {
				fmt.Fprintf(tw, "%v\t", "")
				continue
			}
			fmt.Fprintf(tw, "%v\t", v[i])
		}
		fmt.Fprintf(tw, "\n")
	}
	// close the grid with more '-'
	for _, h := range headers {
		fmt.Fprintf(tw, "%v\t", fs(h))
	}
	fmt.Fprintf(tw, "\n")
	// calculate width and print table from buffer
	tw.Flush()
}
