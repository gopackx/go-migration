package commands

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// confirm prompts the user with the given message and reads a response from
// the provided reader. It returns true if the user enters "y" or "yes"
// (case-insensitive), and false for any other input.
func confirm(in io.Reader, out io.Writer, message string) (bool, error) {
	fmt.Fprintf(out, "%s [y/N]: ", message)

	reader := bufio.NewReader(in)
	line, err := reader.ReadString('\n')
	if err != nil {
		return false, err
	}

	input := strings.TrimSpace(strings.ToLower(line))
	return input == "y" || input == "yes", nil
}
