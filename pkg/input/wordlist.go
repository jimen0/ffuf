package input

import (
	"bufio"
	"bytes"
	"io"
	"os"
	"regexp"

	"github.com/ffuf/ffuf/pkg/ffuf"
)

type WordlistInput struct {
	config   *ffuf.Config
	data     [][]byte
	position int
	keyword  string
}

func NewWordlistInput(keyword string, value string, conf *ffuf.Config) (*WordlistInput, error) {
	wl := &WordlistInput{
		keyword: keyword,
		config:  conf,
	}

	var r io.ReadCloser
	if value == "-" {
		r = os.Stdin
	} else {
		valid, err := wl.validFile(value)
		if err != nil || !valid {
			return wl, err
		}

		r, err = os.Open(value)
		if err != nil {
			return wl, err
		}
	}
	defer r.Close()

	return wl, wl.read(r)
}

//Position will return the current position in the input list
func (w *WordlistInput) Position() int {
	return w.position
}

//ResetPosition resets the position back to beginning of the wordlist.
func (w *WordlistInput) ResetPosition() {
	w.position = 0
}

//Keyword returns the keyword assigned to this InternalInputProvider
func (w *WordlistInput) Keyword() string {
	return w.keyword
}

//Next will increment the cursor position, and return a boolean telling if there's words left in the list
func (w *WordlistInput) Next() bool {
	if w.position >= len(w.data) {
		return false
	}
	return true
}

//IncrementPosition will increment the current position in the inputprovider data slice
func (w *WordlistInput) IncrementPosition() {
	w.position += 1
}

//Value returns the value from wordlist at current cursor position
func (w *WordlistInput) Value() []byte {
	return w.data[w.position]
}

//Total returns the size of wordlist
func (w *WordlistInput) Total() int {
	return len(w.data)
}

//validFile checks that the wordlist file exists and can be read
func (w *WordlistInput) validFile(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	f.Close()
	return true, nil
}

// read reads the given reader line by line to a byte slice.
func (w *WordlistInput) read(r io.Reader) error {
	re := regexp.MustCompile(`(?i)%ext%`)

	var (
		data [][]byte
		ok   bool
	)
	reader := bufio.NewScanner(r)
	for reader.Scan() {
		b := append([]byte{}, reader.Bytes()...)
		if w.config.DirSearchCompat && len(w.config.Extensions) > 0 {
			if re.Match(b) {
				for _, ext := range w.config.Extensions {
					content := re.ReplaceAll(b, []byte(ext))
					data = append(data, content)
				}
			} else {
				if w.config.IgnoreWordlistComments {
					b, ok = stripComments(b)
					if !ok {
						continue
					}
				}
				data = append(data, b)
			}
		} else {
			if w.config.IgnoreWordlistComments {
				b, ok = stripComments(b)
				if !ok {
					continue
				}
			}
			data = append(data, b)
			if w.keyword == "FUZZ" && len(w.config.Extensions) > 0 {
				for _, ext := range w.config.Extensions {
					data = append(data, append(b, []byte(ext)...))
				}
			}
		}
	}
	w.data = data
	return reader.Err()
}

// stripComments removes all kind of comments from the word
func stripComments(b []byte) ([]byte, bool) {
	// If the line starts with a # ignoring any space on the left,
	// return blank.
	if bytes.HasPrefix(bytes.TrimLeft(b, " "), []byte("#")) {
		return []byte{}, false
	}

	// If the line has # later after a space, that's a comment.
	// Only send the word upto space to the routine.
	index := bytes.Index(b, []byte(" #"))
	if index == -1 {
		return b, true
	}
	return b[:index], true
}
