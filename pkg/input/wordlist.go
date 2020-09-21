package input

import (
	"bufio"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/ffuf/ffuf/pkg/ffuf"
)

type WordlistInput struct {
	config   *ffuf.Config
	data     [][]byte
	position int
	keyword  string
}

func NewWordlistInput(keyword string, value string, conf *ffuf.Config) (*WordlistInput, error) {
	var wl WordlistInput
	wl.keyword = keyword
	wl.config = conf
	wl.position = 0

	var r io.ReadCloser
	if value == "-" {
		r = os.Stdin
	} else {
		valid, err := wl.validFile(value)
		if err != nil || !valid {
			return &wl, err
		}

		r, err = os.Open(value)
		if err != nil {
			return &wl, err
		}
	}
	defer r.Close()

	return &wl, wl.read(r)
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
		if w.config.DirSearchCompat && len(w.config.Extensions) > 0 {
			text := []byte(reader.Text())
			if re.Match(text) {
				for _, ext := range w.config.Extensions {
					contnt := re.ReplaceAll(text, []byte(ext))
					data = append(data, []byte(contnt))
				}
			} else {
				text := reader.Text()

				if w.config.IgnoreWordlistComments {
					text, ok = stripComments(text)
					if !ok {
						continue
					}
				}
				data = append(data, []byte(text))
			}
		} else {
			text := reader.Text()

			if w.config.IgnoreWordlistComments {
				text, ok = stripComments(text)
				if !ok {
					continue
				}
			}
			data = append(data, []byte(text))
			if w.keyword == "FUZZ" && len(w.config.Extensions) > 0 {
				for _, ext := range w.config.Extensions {
					data = append(data, []byte(text+ext))
				}
			}
		}
	}
	w.data = data
	return reader.Err()
}

// stripComments removes all kind of comments from the word
func stripComments(text string) (string, bool) {
	// If the line starts with a # ignoring any space on the left,
	// return blank.
	if strings.HasPrefix(strings.TrimLeft(text, " "), "#") {
		return "", false
	}

	// If the line has # later after a space, that's a comment.
	// Only send the word upto space to the routine.
	index := strings.Index(text, " #")
	if index == -1 {
		return text, true
	}
	return text[:index], true
}
