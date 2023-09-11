package oviewer

import "github.com/gdamore/tcell/v2"

// setSaveBuffer is a wrapper to move to setSaveBufferMode.
func (root *Root) setSaveBuffer() {
	if root.Doc.seekable {
		root.setMessage("Saving regular files is not supported")
		return
	}
	root.setSaveBufferMode()
}

// setSaveBufferMode sets the inputMode to SaveBuffer.
func (root *Root) setSaveBufferMode() {
	input := root.input
	input.value = ""
	input.cursorX = 0
	input.Event = newSaveBufferEvent(input.SaveBufferCandidate)
}

// saveBufferCandidate returns the candidate to set to default.
func saveBufferCandidate() *candidate {
	return &candidate{
		list: []string{},
	}
}

// eventSaveBuffer represents the mode input mode.
type eventSaveBuffer struct {
	tcell.EventTime
	clist *candidate
	value string
}

// newSaveBufferEvent returns SaveBufferModeEvent.
func newSaveBufferEvent(clist *candidate) *eventSaveBuffer {
	return &eventSaveBuffer{clist: clist}
}

// Mode returns InputMode.
func (e *eventSaveBuffer) Mode() InputMode {
	return SaveBuffer
}

// Prompt returns the prompt string in the input field.
func (e *eventSaveBuffer) Prompt() string {
	return "(Save)file:"
}

// Confirm returns the event when the input is confirmed.
func (e *eventSaveBuffer) Confirm(str string) tcell.Event {
	e.value = str
	e.SetEventNow()
	return e
}

// Up returns strings when the up key is pressed during input.
func (e *eventSaveBuffer) Up(str string) string {
	return e.clist.up()
}

// Down returns strings when the down key is pressed during input.
func (e *eventSaveBuffer) Down(str string) string {
	return e.clist.down()
}
