package pktline

// Utility functions for working with the Git pkt-line format. See
// https://github.com/git/git/blob/master/Documentation/technical/protocol-common.txt

// TODO add tests https://gitlab.com/gitlab-org/gitaly/issues/1340

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
)

const (
	maxPktSize = 0xffff
)

var (
	flush = []byte("0000")
)

// NewScanner returns a bufio.Scanner that splits on Git pktline boundaries
func NewScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, maxPktSize), maxPktSize)
	scanner.Split(pktLineSplitter)
	return scanner
}

// Data returns the packet pkt without its length header. The length
// header is not validated. Returns nil when pkt is a magic packet such
// as '0000'.
func Data(pkt []byte) []byte {
	if len(pkt) == 4 {
		return nil
	}

	return pkt[4:]
}

// IsFlush detects the special flush packet '0000'
func IsFlush(pkt []byte) bool {
	return bytes.Equal(pkt, flush)
}

// WriteString writes a string with pkt-line framing
func WriteString(w io.Writer, str string) (int, error) {
	pktLen := len(str) + 4
	if pktLen > maxPktSize {
		return 0, fmt.Errorf("string too large: %d bytes", len(str))
	}

	return fmt.Fprintf(w, "%04x%s", pktLen, str)
}

// WriteFlush write a pkt flush packet.
func WriteFlush(w io.Writer) error {
	_, err := w.Write(flush)
	return err
}

func pktLineSplitter(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if len(data) < 4 {
		if atEOF && len(data) > 0 {
			return 0, nil, fmt.Errorf("pktLineSplitter: incomplete length prefix on %q", data)
		}
		return 0, nil, nil // want more data
	}

	// We have at least 4 bytes available so we can decode the 4-hex digit
	// length prefix of the packet line.
	pktLength64, err := strconv.ParseInt(string(data[:4]), 16, 0)
	if err != nil {
		return 0, nil, fmt.Errorf("pktLineSplitter: decode length: %v", err)
	}

	// Cast is safe because we requested an int-size number from strconv.ParseInt
	pktLength := int(pktLength64)

	if pktLength < 0 {
		return 0, nil, fmt.Errorf("pktLineSplitter: invalid length: %d", pktLength)
	}

	if pktLength < 4 {
		// Special case: magic empty packet 0000, 0001, 0002 or 0003.
		return 4, data[:4], nil
	}

	if len(data) < pktLength {
		// data contains incomplete packet

		if atEOF {
			return 0, nil, fmt.Errorf("pktLineSplitter: less than %d bytes in input %q", pktLength, data)
		}

		return 0, nil, nil // want more data
	}

	return pktLength, data[:pktLength], nil
}