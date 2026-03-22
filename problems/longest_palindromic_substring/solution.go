package longest_palindromic_substring

import (
	"bufio"
	"log"
	"os"
)

var (
	logger *log.Logger
	logBuf *bufio.Writer
)

func init() {
	logBuf = bufio.NewWriter(os.Stdout)
	logger = log.New(logBuf, "", 0)
}

type Polindrome struct {
	Start int
	End   int
	input string
}

func (p Polindrome) StringSlice() string {
	return p.input[p.Start : p.End+1]
}

func (p Polindrome) Lenght() int {
	return p.End + 1 - p.Start
}

func (p Polindrome) String() string {
	return p.StringSlice()
}

func longestPalindrome(s string) string {
	var longest string

	polindromes := make([]Polindrome, 0)

	if len(s) < 2 {
		return s
	}

	for i := 1; i < len(s); i++ {
		currC := s[i]

		polindromes = append(polindromes, Polindrome{i, i, s})

		bfrI := i - 1
		if bfr, ok := safeGetSymbolByIndex(s, bfrI); ok {

			if bfr == currC {
				new := Polindrome{bfrI, i, s}
				polindromes = append(polindromes, new)

				// logger.Printf("+even polindrome has added %s", new)

			}
		}
	}

	for _, p := range polindromes {
		bfrI := p.Start - 1
		aftI := p.End + 1

		for {
			bfr, bfrOk := safeGetSymbolByIndex(s, bfrI)
			if !bfrOk {
				// logger.Printf("-index %d not found", bfrI)
				break
			}
			after, aftOk := safeGetSymbolByIndex(s, aftI)
			if !aftOk {
				// logger.Printf("-index %d not found", aftI)
				break
			}

			if bfr != after {
				// logger.Printf("!= %c(%d) not equal %c(%d)", bfr, bfrI, after, aftI)
				break
			}

			p.Start = bfrI
			p.End = aftI

			if p.Lenght() < 10 {

				// logger.Printf("+polindrome increased %s: (%d)", p, p.Lenght())
			}

			bfrI--
			aftI++

		}

		if p.Lenght() > len(longest) {
			longest = p.String()

			if p.Lenght() < 10 {
				// logger.Printf("=longest assigned: %s (%d)", longest, len(longest))
			}
		}

	}

	// logger.Printf("<< longest: %s (%d)", longest, len(longest))

	return longest
}

func safeGetSymbolByIndex(s string, idx int) (byte, bool) {
	if idx < 0 || idx > len(s)-1 {
		return 0, false
	}

	return s[idx], true
}

func Flushlogs() error {
	return logBuf.Flush()
}

func ResetLogs() {
	logBuf.Reset(os.Stdout)
}
