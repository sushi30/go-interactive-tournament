package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// compareRequest is sent by the sorter to the UI; the UI must send the boolean
// response on Resp (true => prefer A, false => prefer B).
type compareRequest struct {
	A, B string
	Resp chan bool
}

// RunTUI launches a Bubbletea TUI that displays two items side-by-side as "cards"
// and lets the user choose using the left/right arrow keys (or 'h'/'l').
// It drives a goroutine that runs the merge-sort and supplies comparison requests
// to the UI via compCh. When sorting is done the final ranking is printed.
func RunTUI(items []string) []string {
	compCh := make(chan compareRequest)
	sortDoneCh := make(chan struct{})

	// channel where sorted result will be delivered
	doneCh := make(chan []string)

	// start sorter in background; it will send compareRequest into compCh
	go func() {
		sorted := mergeSortWithCmp(items, func(a, b string) bool {
			req := compareRequest{A: a, B: b, Resp: make(chan bool)}
			compCh <- req
			return <-req.Resp
		})
		// deliver sorted result first
		doneCh <- sorted
		// signal done to pump so it can quit the UI automatically
		close(sortDoneCh)
		// then close compCh to unblock any readers
		close(compCh)
	}()

	// Initialize and run Bubbletea program
	p := tea.NewProgram(initialModel(compCh))

	// ensure UI quits when sorter finishes and capture sorted result
	sortedReady := make(chan []string, 1)
	go func() {
		s := <-doneCh
		// deliver result to be returned after program exits
		sortedReady <- s
		// request UI to quit (in case pump misses it)
		p.Send(quitMsg{})
	}()

	// pump compare requests into the program using Program.Send.
	// We also listen for the sorter-done signal and send quit when sorting finishes.
	go func() {
		for {
			select {
			case req, ok := <-compCh:
				if !ok {
					// channel closed; ensure quit is sent and exit pump
					p.Send(quitMsg{})
					return
				}
				p.Send(cmpRequestMsg(req))
			case <-sortDoneCh:
				// sorter finished; request UI to quit
				p.Send(quitMsg{})
				return
			}
		}
	}()

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to run TUI: %v\n", err)
		os.Exit(1)
	}

	// Wait for sorter result and return it
	sorted := <-sortedReady
	return sorted
}

// ----------------------
// Bubbletea model
// ----------------------

type model struct {
	// channel receiving compareRequest from sorter
	compCh <-chan compareRequest

	// the currently displayed pair (when non-empty)
	a, b string

	// the current active request to answer; nil when idle
	req chan bool

	// a small status message
	status string
}

func initialModel(compCh <-chan compareRequest) model {
	return model{compCh: compCh, status: "Waiting for first comparison..."}
}

// Messages
type cmpRequestMsg compareRequest
type quitMsg struct{}
type tickMsg struct{}

func (m model) Init() tea.Cmd {
	// No-op; RunTUI pumps compare requests into the program via p.Send.
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case cmpRequestMsg:
		req := compareRequest(msg)
		m.a = req.A
		m.b = req.B
		m.req = req.Resp
		m.status = "Use ← / → (or h / l) to choose. Enter q to quit."
		// after handling this request, schedule another read for the next request
		return m, m.Init()
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if m.req != nil {
				m.req <- true
				m.req = nil
				m.status = "Sent choice: left"
			}
			return m, nil
		case "right", "l":
			if m.req != nil {
				m.req <- false
				m.req = nil
				m.status = "Sent choice: right"
			}
			return m, nil
		case "q", "ctrl+c":
			// quit the program immediately
			return m, tea.Quit
		}
	case quitMsg:
		return m, tea.Quit
	case tea.WindowSizeMsg:
		// ignore for now, could be used to layout cards
	case nil:
		// ignore
	}
	return m, nil
}

func (m model) View() string {
	out := "\nInteractive Tournament — choose which item you prefer\n\n"
	if m.a == "" && m.b == "" {
		out += fmt.Sprintf("%s\n", m.status)
		return out
	}
	// Simple side-by-side "cards" using fixed-width formatting
	out += fmt.Sprintf("  ← %-30s    %-30s →\n\n", m.a, m.b)
	out += fmt.Sprintf("  %s\n\n", m.status)
	return out
}

// ----------------------
// sorter with comparator
// ----------------------

// mergeSortWithCmp is a standard merge-sort that uses cmp(a,b) to decide
// whether a should come before b (cmp returns true if a preferred over b).
func mergeSortWithCmp(items []string, cmp func(a, b string) bool) []string {
	if len(items) <= 1 {
		out := make([]string, len(items))
		copy(out, items)
		return out
	}
	mid := len(items) / 2
	left := mergeSortWithCmp(items[:mid], cmp)
	right := mergeSortWithCmp(items[mid:], cmp)
	return mergeWithCmp(left, right, cmp)
}

func mergeWithCmp(left, right []string, cmp func(a, b string) bool) []string {
	i, j := 0, 0
	out := make([]string, 0, len(left)+len(right))
	for i < len(left) && j < len(right) {
		if cmp(left[i], right[j]) {
			out = append(out, left[i])
			i++
		} else {
			out = append(out, right[j])
			j++
		}
	}
	for i < len(left) {
		out = append(out, left[i])
		i++
	}
	for j < len(right) {
		out = append(out, right[j])
		j++
	}
	return out
}
