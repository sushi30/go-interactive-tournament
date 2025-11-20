package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Interactive tournament-style (merge) sort.
// The program reads a list of items from a file, command-line arguments,
// or interactively from the user, then sorts them by repeatedly asking
// pairwise "which do you prefer?" questions and producing a ranked list.

var reader *bufio.Reader

func main() {
	reader = bufio.NewReader(os.Stdin)

	filePath := flag.String("file", "", "Path to a file with one item per line")
	tuiMode := flag.Bool("tui", false, "Launch interactive TUI (Bubbletea)")
	outPath := flag.String("o", "", "Write final ranking to a file")
	flag.Parse()

	items := []string{}

	// Order of preference for input:
	// 1) -file
	// 2) command-line args (positional)
	// 3) interactive prompt (enter items one-per-line, blank line to finish)
	if *filePath != "" {
		data, err := os.ReadFile(*filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read file: %v\n", err)
			os.Exit(1)
		}
		for _, line := range strings.Split(string(data), "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				items = append(items, line)
			}
		}
	} else if flag.NArg() > 0 {
		for _, a := range flag.Args() {
			a = strings.TrimSpace(a)
			if a != "" {
				items = append(items, a)
			}
		}
	} else {
		fmt.Println("Enter items to sort, one per line. Submit an empty line when done:")
		for {
			fmt.Print("> ")
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "read error: %v\n", err)
				os.Exit(1)
			}
			line = strings.TrimSpace(line)
			if line == "" {
				break
			}
			items = append(items, line)
		}
	}

	if len(items) == 0 {
		fmt.Fprintln(os.Stderr, "no items provided; exiting")
		os.Exit(1)
	}

	if *tuiMode {
		sorted := RunTUI(items)
		// Print final ranking to stdout (RunTUI no longer prints)
		fmt.Println("\nFinal ranking (best -> worst):")
		for i, it := range sorted {
			fmt.Printf("%d. %s\n", i+1, it)
		}
		if *outPath != "" {
			if err := writeResultToFile(sorted, *outPath); err != nil {
				fmt.Fprintf(os.Stderr, "failed to write output: %v\n", err)
				os.Exit(1)
			}
		}
		return
	}

	fmt.Printf("\nGot %d items. We'll ask pairwise questions to rank them.\n", len(items))
	fmt.Println("On each prompt enter 1 or 2 to choose the item you prefer. Enter q to quit.\n")

	sorted := interactiveMergeSort(items)

	fmt.Println("\nFinal ranking (best -> worst):")
	for i, it := range sorted {
		fmt.Printf("%d. %s\n", i+1, it)
	}
	if *outPath != "" {
		if err := writeResultToFile(sorted, *outPath); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write output file: %v\n", err)
			os.Exit(1)
		}
	}
}

// interactiveMergeSort performs a standard merge sort but asks the user
// to compare elements during merge instead of using a deterministic comparison.
func interactiveMergeSort(items []string) []string {
	if len(items) <= 1 {
		// Make a copy to avoid aliasing
		out := make([]string, len(items))
		copy(out, items)
		return out
	}
	mid := len(items) / 2
	left := interactiveMergeSort(items[:mid])
	right := interactiveMergeSort(items[mid:])
	return interactiveMerge(left, right)
}

// interactiveMerge merges two sorted slices by asking the user which item they prefer.
func interactiveMerge(left, right []string) []string {
	i, j := 0, 0
	out := make([]string, 0, len(left)+len(right))
	for i < len(left) && j < len(right) {
		// Ask user which item they prefer: left[i] or right[j]
		if askPreference(left[i], right[j]) {
			out = append(out, left[i])
			i++
		} else {
			out = append(out, right[j])
			j++
		}
	}
	// append remaining
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

// askPreference returns true if the user prefers a over b.
// It prompts until it receives a valid answer: 1 => a, 2 => b, q => quit.
func askPreference(a, b string) bool {
	for {
		fmt.Println("Which do you prefer?")
		fmt.Printf("  1) %s\n", a)
		fmt.Printf("  2) %s\n", b)
		fmt.Print("Enter 1 or 2 (or q to quit): ")

		resp, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintf(os.Stderr, "read error: %v\n", err)
			os.Exit(1)
		}
		resp = strings.TrimSpace(resp)
		switch strings.ToLower(resp) {
		case "1", "a":
			return true
		case "2", "b":
			return false
		case "q", "quit", "exit":
			fmt.Println("Quitting.")
			os.Exit(0)
		default:
			fmt.Println("Invalid input; please enter 1 or 2 (or q to quit).")
		}
	}
}

func writeResultToFile(sorted []string, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for i, it := range sorted {
		if _, err := fmt.Fprintf(w, "%d. %s\n", i+1, it); err != nil {
			return err
		}
	}
	return w.Flush()
}
