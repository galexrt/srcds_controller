// Originally taken from https://github.com/c-bata/go-prompt project
// Their code is licensed under MIT
// For more information see their git repository.

package linehistory

import "strings"

// History stores the texts that are entered.
type History struct {
	histories []string
	tmp       []string
	selected  int
}

// Add to add text in history.
func (h *History) Add(input string) {
	h.histories = append(h.histories, input)
	h.Clear()
}

// Clear to clear the history.
func (h *History) Clear() {
	h.tmp = make([]string, len(h.histories))
	for i := range h.histories {
		h.tmp[i] = h.histories[i]
	}
	h.tmp = append(h.tmp, "")
	h.selected = len(h.tmp) - 1
}

// Older saves a buffer of current line and get a buffer of previous line by up-arrow.
// The changes of line buffers are stored until new history is created.
func (h *History) Older(buf string) (new string, changed bool) {
	if len(h.tmp) == 1 || h.selected == 0 {
		return buf, false
	}
	h.tmp[h.selected] = buf

	h.selected--
	return h.tmp[h.selected], true
}

// Newer saves a buffer of current line and get a buffer of next line by up-arrow.
// The changes of line buffers are stored until new history is created.
func (h *History) Newer(buf string) (new string, changed bool) {
	if h.selected >= len(h.tmp)-1 {
		return buf, false
	}
	h.tmp[h.selected] = buf

	h.selected++
	return h.tmp[h.selected], true
}

// Dump return all history entries as string
func (h *History) Dump() string {
	return strings.Join(h.histories, "\n")
}

// NewHistory returns new history object.
func NewHistory() *History {
	return &History{
		histories: []string{},
		tmp:       []string{""},
		selected:  0,
	}
}
