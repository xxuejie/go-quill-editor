package editor

import (
	"bytes"
	"fmt"
)

// Corresponds to Addr in https://github.com/9fans/plan9port/blob/4650064aa757c217fa72f8819a2cf67c689bcdef/src/cmd/acme/edit.h#L16
type Addr struct {
	t byte

	// Either re or left can exist
	re   string
	left *Addr // left side of , and ;

	num  int64
	next *Addr // right side of , and ;
}

type Cmd struct {
	addr *Addr
	re   string

	// One of cmd, text and mtaddr can exist
	cmd    *Cmd   // target of x, g, {, etc.
	text   string // text of a, c, i; rhs of s
	mtaddr *Addr  // address for m, t

	next *Cmd // elements in {
	num  int64
	flag uint16
	cmdc uint16 // command character
}

type textRange struct {
	q0 int64
	q1 int64
}

type cmdScanner struct {
	c string
	i int
}

func newCmdScanner(c string) *cmdScanner {
	return &cmdScanner{
		c: c,
		i: 0,
	}
}

func (s *cmdScanner) read() (byte, bool) {
	if s.i < len(s.c) {
		s.i += 1
		return s.c[s.i-1], true
	}
	return 0, false
}

func (s *cmdScanner) unread() {
	if s.i > 0 {
		s.i -= 1
	}
}

func (s *cmdScanner) readNum(processSign bool) int64 {
	n := int64(0)
	sign := int64(1)
	if processSign {
		ch, success := s.peek()
		if success && ch == '-' {
			sign = -1
			s.read()
		}
	}
	ch, success := s.peek()
	if (!success) || (ch < '0') || (ch > '9') {
		return sign
	}
	for {
		ch, success = s.peek()
		if (!success) || (ch < '0') || (ch > '9') {
			break
		}
		n = n*10 + int64(ch-'0')
		s.read()
	}
	return sign * n
}

func (s *cmdScanner) readRegexp(delimiter byte) (string, error) {
	buffer := make([]byte, 0)
	var c byte
	var success bool
	for {
		c, success = s.read()
		if !success {
			return "", fmt.Errorf("Unexpected regexp ending!")
		}
		if c == '\\' {
			if nextc, success2 := s.peek(); success2 {
				if nextc == delimiter {
					c, _ = s.read()
				} else if nextc == '\\' {
					buffer = append(buffer, c)
					c, success = s.read()
					if !success {
						return "", fmt.Errorf("Unexpected regexp ending!")
					}
				}
			}
		} else if c == delimiter || c == '\n' {
			break
		}
		buffer = append(buffer, c)
	}
	if success && c != delimiter {
		s.unread()
	}
	if len(buffer) == 0 {
		return "", fmt.Errorf("No regular expression defined!")
	}
	return string(buffer), nil
}

func (s *cmdScanner) readRhs(delimiter byte, cmd byte) (string, error) {
	buffer := make([]byte, 0)
	var c byte
	var success bool
	for {
		c, success = s.read()
		if (!success) || c == delimiter || c == '\n' {
			break
		}
		if c == '\\' {
			c, success = s.read()
			if !success {
				return "", fmt.Errorf("Bad right hand side!")
			}
			if c == '\n' {
				s.unread()
				c = '\\'
			} else if c == 'n' {
				c = '\n'
			} else if c != delimiter && (cmd == 's' || c != '\\') {
				buffer = append(buffer, '\\')
			}
		}
		buffer = append(buffer, c)
	}
	if success {
		s.unread()
	}
	return string(buffer), nil
}

func (s *cmdScanner) readText() (string, error) {
	buffer := make([]byte, 0)
	c, success := s.peekSkipBlank()
	if success && c == '\n' {
		s.read()
		for {
			line := make([]byte, 0)
			for {
				c, success = s.read()
				if (!success) || c == '\n' {
					break
				}
				line = append(line, c)
			}
			line = append(line, '\n')
			if !success {
				buffer = append(buffer, line...)
				goto Return
			}
			if len(line) == 2 && line[0] == '.' && line[1] == '\n' {
				break
			}
			buffer = append(buffer, line...)
		}
	} else {
		delimiter, _ := s.read()
		if err := checkOkDelimiter(delimiter); err != nil {
			return "", err
		}
		str, err := s.readRhs(delimiter, 'a')
		if err != nil {
			return "", err
		}
		if nc, ns := s.peek(); ns && nc == delimiter {
			s.read()
		}
		if err := s.assertLineEnd(); err != nil {
			return "", err
		}
		return str, nil
	}
Return:
	return string(buffer), nil
}

func (s *cmdScanner) readToken(terminatingTokens []byte) (string, error) {
	buffer := make([]byte, 0)
	var c byte
	var success bool
	for c, success = s.peek(); success && (c == ' ' || c == '\t'); c, success = s.peek() {
		s.read()
		buffer = append(buffer, c)
	}
	for c, success = s.read(); success && (!isToken(c, terminatingTokens)); c, success = s.read() {
		buffer = append(buffer, c)
	}
	if c != '\n' {
		if err := s.assertLineEnd(); err != nil {
			return "", err
		}
	}
	return string(buffer), nil
}

func (s *cmdScanner) assertLineEnd() error {
	s.peekSkipBlank()
	c, success := s.read()
	if !success {
		return fmt.Errorf("Newline expected but no char is provided!")
	}
	if c != '\n' {
		return fmt.Errorf("Newline expected (saw %c)", c)
	}
	return nil
}

func (s *cmdScanner) peek() (byte, bool) {
	if s.i < len(s.c) {
		return s.c[s.i], true
	}
	return 0, false
}

func (s *cmdScanner) peekSkipBlank() (byte, bool) {
	for s.i < len(s.c) && (s.c[s.i] == ' ' || s.c[s.i] == '\t') {
		s.i += 1
	}
	return s.peek()
}

func parseSimpleAddr(s *cmdScanner) (*Addr, error) {
	var addr Addr
	var err error
	ch, success := s.peekSkipBlank()
	if !success {
		return nil, nil
	}
	switch ch {
	case '#':
		addr.t, _ = s.read()
		addr.num = s.readNum(false)
	case '0':
		fallthrough
	case '1':
		fallthrough
	case '2':
		fallthrough
	case '3':
		fallthrough
	case '4':
		fallthrough
	case '5':
		fallthrough
	case '6':
		fallthrough
	case '7':
		fallthrough
	case '8':
		fallthrough
	case '9':
		addr.t = 'l'
		addr.num = s.readNum(false)
	case '/':
		fallthrough
	case '?':
		fallthrough
	case '"':
		addr.t, _ = s.read()
		addr.re, err = s.readRegexp(addr.t)
		if err != nil {
			return nil, err
		}
	case '.':
		fallthrough
	case '$':
		fallthrough
	case '+':
		fallthrough
	case '-':
		fallthrough
	case '\'':
		addr.t, _ = s.read()
	default:
		return nil, nil
	}
	addr.next, err = parseSimpleAddr(s)
	if err != nil {
		return nil, err
	}
	if addr.next != nil {
		switch addr.next.t {
		case '.':
			fallthrough
		case '$':
			fallthrough
		case '\'':
			if addr.t != '"' {
				return nil, fmt.Errorf("Bad address syntax!")
			}
		case '"':
			return nil, fmt.Errorf("Bad address syntax!")
		case 'l':
			fallthrough
		case '#':
			if addr.t == '"' {
				break
			}
			fallthrough
		case '/':
			fallthrough
		case '?':
			if addr.t != '+' && addr.t != '-' {
				// Insert missing '+'
				nap := Addr{
					t:    '+',
					next: addr.next,
				}
				addr.next = &nap
			}
		case '+':
			fallthrough
		case '-':
		default:
			return nil, fmt.Errorf("Simple address error!")
		}
	}
	return &addr, nil
}

func parseCompoundAddr(s *cmdScanner) (*Addr, error) {
	var addr Addr
	var err error
	var success bool
	addr.left, err = parseSimpleAddr(s)
	if err != nil {
		return nil, err
	}
	addr.t, success = s.peekSkipBlank()
	if (!success) || (addr.t != ',' && addr.t != ';') {
		return addr.left, nil
	}
	s.read()
	next, err := parseCompoundAddr(s)
	if err != nil {
		return nil, err
	}
	addr.next = next
	if next != nil && (next.t == ',' || next.t == ';') && next.left == nil {
		return nil, fmt.Errorf("Bad compound address syntax!")
	}
	return &addr, nil
}

func innerParseCmd(s *cmdScanner, nest int) (*Cmd, error) {
	var cmd Cmd
	var err error
	cmd.addr, err = parseCompoundAddr(s)
	if err != nil {
		return nil, err
	}
	if _, success := s.peekSkipBlank(); !success {
		return nil, nil
	}
	c, success := s.read()
	if !success {
		return nil, nil
	}
	cmd.cmdc = uint16(c)
	if c == 'c' {
		nc, ns := s.peek()
		if ns && nc == 'd' {
			s.read()
			cmd.cmdc = uint16('c') | 0x100 // command cd
		}
	}
	ct := cmdLookup(cmd.cmdc)
	if ct != nil {
		if cmd.cmdc == uint16('\n') {
			goto Return
		}
		if ct.defaddr == defAddrNo && cmd.addr != nil {
			return nil, fmt.Errorf("Command takes no address!")
		}
		if ct.count > 0 {
			cmd.num = s.readNum(ct.count > 1)
		}
		if ct.regexp {
			nc, ns := s.peek()
			if (ct.cmdc != uint16('x') && ct.cmdc != uint16('X')) ||
				((!ns) || (nc != ' ' && nc != '\t' && nc != '\n')) {
				s.peekSkipBlank()
				c, success := s.read()
				if (!success) || (c == '\n') {
					return nil, fmt.Errorf("No address!")
				}
				if err = checkOkDelimiter(c); err != nil {
					return nil, err
				}
				cmd.re, err = s.readRegexp(c)
				if err != nil {
					return nil, err
				}
				if ct.cmdc == uint16('s') {
					cmd.text, err = s.readRhs(c, 's')
					if err != nil {
						return nil, err
					}
					if nc, ns := s.peek(); ns && nc == c {
						s.read()
						if nc, ns := s.peek(); ns && nc == 'g' {
							s.read()
							cmd.flag = uint16(nc)
						}
					}
				}
			}
		}
		if ct.addr {
			cmd.mtaddr, err = parseSimpleAddr(s)
			if err != nil {
				return nil, err
			}
			if cmd.mtaddr == nil {
				return nil, fmt.Errorf("Bad address!")
			}
		}
		if ct.defcmd != 0 {
			if nc, ns := s.peekSkipBlank(); ns && nc == '\n' {
				s.read()
				cmd.cmd = &Cmd{
					cmdc: uint16(ct.defcmd),
				}
			} else {
				newcmd, err := innerParseCmd(s, nest)
				if err != nil {
					return nil, err
				}
				if newcmd == nil {
					return nil, fmt.Errorf("Defcmd!")
				}
				cmd.cmd = newcmd
			}
		} else if ct.text {
			cmd.text, err = s.readText()
			if err != nil {
				return nil, err
			}
		} else if ct.token != nil {
			cmd.text, err = s.readToken(ct.token)
			if err != nil {
				return nil, err
			}
		} else {
			if err = s.assertLineEnd(); err != nil {
				return nil, err
			}
		}
	} else {
		switch cmd.cmdc {
		case '{':
			var currentCmd *Cmd
			for {
				if nc, ns := s.peekSkipBlank(); ns && nc == '\n' {
					s.read()
				}
				ncmd, err := innerParseCmd(s, nest+1)
				if err != nil {
					return nil, err
				}
				if ncmd == nil {
					break
				}
				if currentCmd != nil {
					currentCmd.next = ncmd
				} else {
					cmd.cmd = ncmd
				}
				currentCmd = ncmd
			}
		case '}':
			if err = s.assertLineEnd(); err != nil {
				return nil, err
			}
			if nest == 0 {
				return nil, fmt.Errorf("Right brace with no left brace!")
			}
			return nil, nil
		default:
			return nil, fmt.Errorf("Unknown command %c(0x%x)", byte(cmd.cmdc), cmd.cmdc)
		}
	}
Return:
	return &cmd, nil
}

func checkOkDelimiter(c byte) error {
	if c == '\\' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
		(c >= '0' && c <= '9') {
		return fmt.Errorf("Bad delimiter %c!", c)
	}
	return nil
}

func isToken(c byte, tokens []byte) bool {
	return bytes.IndexByte(tokens, c) != -1
}
