package oviewer

import "github.com/gdamore/tcell/v2"

// setSearchMode sets the inputMode to Search.
func (root *Root) setSearchMode() {
	input := root.input
	input.value = ""
	input.cursorX = 0
	input.Event = newSearchEvent(input.SearchCandidate)
	root.OriginPos = root.Doc.topLN
}

// searchCandidate returns the candidate to set to default.
func searchCandidate() *candidate {
	return &candidate{
		list: []string{},
	}
}

// eventInputSearch represents the search input mode.
type eventInputSearch struct {
	tcell.EventTime
	clist *candidate
	value string
}

// newSearchEvent returns SearchInput.
func newSearchEvent(clist *candidate) *eventInputSearch {
	return &eventInputSearch{
		value:     "",
		clist:     clist,
		EventTime: tcell.EventTime{},
	}
}

// Mode returns InputMode.
func (e *eventInputSearch) Mode() InputMode {
	return Search
}

// Prompt returns the prompt string in the input field.
func (e *eventInputSearch) Prompt() string {
	return "/"
}

// Confirm returns the event when the input is confirmed.
func (e *eventInputSearch) Confirm(str string) tcell.Event {
	e.value = str
	e.clist.list = toLast(e.clist.list, str)
	e.clist.p = 0
	e.SetEventNow()
	return e
}

// Up returns strings when the up key is pressed during input.
func (e *eventInputSearch) Up(str string) string {
	e.clist.list = toAddLast(e.clist.list, str)
	return e.clist.up()
}

// Down returns strings when the down key is pressed during input.
func (e *eventInputSearch) Down(str string) string {
	e.clist.list = toAddTop(e.clist.list, str)
	return e.clist.down()
}

// setBackSearchMode sets the inputMode to Backsearch.
func (root *Root) setBackSearchMode() {
	input := root.input
	input.value = ""
	input.cursorX = 0
	input.Event = newBackSearchEvent(input.SearchCandidate)
	root.OriginPos = root.Doc.topLN
}

// eventInputBackSearch represents the back search input mode.
type eventInputBackSearch struct {
	tcell.EventTime
	clist *candidate
	value string
}

// newBackSearchEvent returns backSearchEvent.
func newBackSearchEvent(clist *candidate) *eventInputBackSearch {
	return &eventInputBackSearch{clist: clist}
}

func (e *eventInputBackSearch) Mode() InputMode {
	return Backsearch
}

// Prompt returns the prompt string in the input field.
func (e *eventInputBackSearch) Prompt() string {
	return "?"
}

// Confirm returns the event when the input is confirmed.
func (e *eventInputBackSearch) Confirm(str string) tcell.Event {
	e.value = str
	e.clist.list = toLast(e.clist.list, str)
	e.clist.p = 0
	e.SetEventNow()
	return e
}

// Up returns strings when the up key is pressed during input.
func (e *eventInputBackSearch) Up(str string) string {
	e.clist.list = toAddLast(e.clist.list, str)
	return e.clist.up()
}

// Down returns strings when the down key is pressed during input.
func (e *eventInputBackSearch) Down(str string) string {
	e.clist.list = toAddTop(e.clist.list, str)
	return e.clist.down()
}

// searchCandidates returns the list of search candidates.
func (root *Root) searchCandidates(n int) []string {
	listLen := len(root.input.SearchCandidate.list)
	start := max(0, listLen-n)
	return root.input.SearchCandidate.list[start:listLen]
}
