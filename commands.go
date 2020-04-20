package editor

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
)

type defaultAddress byte

const (
	defAddrNo = iota
	defAddrDot
	defAddrAll
)

type innerContext struct {
	File    *innerFile
	Printer io.Writer
}

func newInnerContext(context Context) (innerContext, error) {
	innerFile, err := newInnerFile(context.File)
	if err != nil {
		return innerContext{}, err
	}
	return innerContext{
		File:    innerFile,
		Printer: context.Printer,
	}, nil
}

type cmdtab struct {
	cmdc    uint16                                    // command character
	text    bool                                      // takes a textual argument?
	regexp  bool                                      // takes a regular expression?
	addr    bool                                      // takes an address (m or t)?
	defcmd  byte                                      // default command; 0 means none
	defaddr defaultAddress                            // default address
	count   byte                                      // takes a count
	token   []byte                                    // takes text terminated by one of these
	fn      func(context innerContext, cmd Cmd) error // function to call
}

var (
	lineTokens = []byte{'\n'}
	wordTokens = []byte{' ', '\t', '\n'}
)

var (
	posnLine      = 0
	posnChars     = 1
	posnLineChars = 2
)

var cmdtabs []cmdtab

func init() {
	cmdtabs = []cmdtab{
		{
			cmdc:    '\n',
			text:    false,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      nlCmd,
		},
		{
			cmdc:    'a',
			text:    true,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      aCmd,
		},
		{
			cmdc:    'c',
			text:    true,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      cCmd,
		},
		{
			cmdc:    'd',
			text:    false,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      dCmd,
		},
		{
			cmdc:    'g',
			text:    false,
			regexp:  true,
			addr:    false,
			defcmd:  'p',
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      gCmd,
		},
		{
			cmdc:    'i',
			text:    true,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      iCmd,
		},
		{
			cmdc:    'm',
			text:    false,
			regexp:  false,
			addr:    true,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      mCmd,
		},
		{
			cmdc:    'p',
			text:    false,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      pCmd,
		},
		{
			cmdc:    's',
			text:    false,
			regexp:  true,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   1,
			token:   nil,
			fn:      sCmd,
		},
		{
			cmdc:    't',
			text:    false,
			regexp:  false,
			addr:    true,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      tCmd,
		},
		{
			cmdc:    'v',
			text:    false,
			regexp:  true,
			addr:    false,
			defcmd:  'p',
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      gCmd,
		},
		{
			cmdc:    'x',
			text:    false,
			regexp:  true,
			addr:    false,
			defcmd:  'p',
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      xCmd,
		},
		{
			cmdc:    'y',
			text:    false,
			regexp:  true,
			addr:    false,
			defcmd:  'p',
			defaddr: defAddrDot,
			count:   0,
			token:   nil,
			fn:      xCmd,
		},
		{
			cmdc:    '=',
			text:    false,
			regexp:  false,
			addr:    false,
			defcmd:  0,
			defaddr: defAddrDot,
			count:   0,
			token:   lineTokens,
			fn:      eqCmd,
		},
	}
}

func cmdLookup(cmdc uint16) *cmdtab {
	for _, cmdtab := range cmdtabs {
		if cmdtab.cmdc == cmdc {
			return &cmdtab
		}
	}
	return nil
}

func cmdExec(c Cmd, context innerContext) error {
	ct := cmdLookup(c.cmdc)
	if ct != nil && ct.defaddr != defAddrNo {
		if c.addr == nil && c.cmdc != '\n' {
			c.addr = &Addr{
				t: '.',
			}
			if ct.defaddr == defAddrAll {
				c.addr.t = '*'
			}
		} else if c.addr != nil && c.addr.t == '"' && c.addr.next == nil && c.cmdc != '\n' {
			c.addr.next = &Addr{
				t: '.',
			}
			if ct.defaddr == defAddrAll {
				c.addr.next.t = '*'
			}
		}
		if c.addr != nil {
			a, err := cmdAddress(c.addr, context, 0)
			if err != nil {
				return err
			}
			context.File.Select(a[0], a[1])
		}
	}
	switch c.cmdc {
	case '{':
		if c.addr != nil {
			a, err := cmdAddress(c.addr, context, 0)
			if err != nil {
				return err
			}
			context.File.Select(a[0], a[1])
		}
		q0, q1 := context.File.Dot()
		for cc := c.cmd; cc != nil; cc = cc.next {
			context.File.Select(q0, q1)
			err := cmdExec(*cc, context)
			if err != nil {
				return err
			}
		}
	default:
		if ct == nil {
			return fmt.Errorf("Unknown command %c(0x%x) in cmdexec", byte(c.cmdc), c.cmdc)
		}
		return ct.fn(context, c)
	}
	return nil
}

func cmdAddress(addr *Addr, context innerContext, sign int) ([]int64, error) {
	a0, a1 := context.File.Dot()
	result := []int64{a0, a1}
	var err error
	for addr != nil {
		switch addr.t {
		case '#':
			if sign == 0 {
				result[0] = addr.num
				result[1] = addr.num
			} else if sign < 0 {
				result[0] -= addr.num
				result[1] = result[0]
			} else if sign > 0 {
				result[1] += addr.num
				result[0] = result[1]
			}
			if result[0] > context.File.Len() {
				return nil, fmt.Errorf("Address out of range!")
			}
		case 'l':
			location, err := extractLineAddress(context, addr.num, sign, result)
			if err != nil {
				return nil, err
			}
			result = location
		case '.':
			// No action needed
		case '$':
			l := context.File.Len()
			result[0], result[1] = l, l
		case '\'':
			return nil, fmt.Errorf("Can't handle '")
		case '?':
			sign = -sign
			if sign == 0 {
				sign = -1
			}
			fallthrough
		case '/':
			// TODO: adding wrapping support later
			start := result[1]
			end := context.File.Len()
			if sign < 0 {
				start = 0
				end = result[0]
			}
			location, err := regexpSearch(addr.re, context, start, end, sign)
			if err != nil {
				return nil, err
			}
			if location == nil {
				return nil, fmt.Errorf("No match for regexp")
			}
			result = location
		case '"':
			return nil, fmt.Errorf("Implement \" later")
		case '*':
			result[0], result[1] = 0, context.File.Len()
		case ',':
			fallthrough
		case ';':
			// TODO: deal with file selection later
			a1 := []int64{0, 0}
			if addr.left != nil {
				a1, err = cmdAddress(addr.left, context, 0)
				if err != nil {
					return nil, err
				}
			}
			if addr.t == ';' {
				result = a1
				context.File.Select(a1[0], a1[1])
			}
			l := context.File.Len()
			a2 := []int64{l, l}
			if addr.next != nil {
				a2, err = cmdAddress(addr.next, context, 0)
				if err != nil {
					return nil, err
				}
			}
			result[0], result[1] = a1[0], a2[1]
			if result[1] < result[0] {
				return nil, fmt.Errorf("Addresses out of order")
			}
			return result, nil
		case '+':
			fallthrough
		case '-':
			sign = 1
			if addr.t == '-' {
				sign = -1
			}
			if addr.next == nil || addr.next.t == '+' || addr.next.t == '-' {
				result, err = extractLineAddress(context, 1, sign, result)
				if err != nil {
					return nil, err
				}
			}
		default:
			return nil, fmt.Errorf("Invalid addresss type %c when setting dot!", addr.t)
		}
		addr = addr.next
	}
	return result, nil
}

func nlCmd(context innerContext, cmd Cmd) error {
	q0, q1 := context.File.Dot()
	addr := []int64{q0, q1}
	var err error
	if cmd.addr == nil {
		addr, err = extractLineAddress(context, 0, -1, []int64{q0, q1})
		if err != nil {
			return err
		}
		a, err := extractLineAddress(context, 0, 1, []int64{q0, q1})
		if err != nil {
			return err
		}
		addr[1] = a[1]
		if addr[0] == q0 && addr[1] == q1 {
			addr, err = extractLineAddress(context, 1, 1, []int64{q0, q1})
			if err != nil {
				return err
			}
		}
	}
	context.File.Select(addr[0], addr[1])
	return nil
}

func aCmd(context innerContext, cmd Cmd) error {
	_, q1 := context.File.Dot()
	l := int64(0)
	if len(cmd.text) > 0 {
		data := []byte(cmd.text)
		l = context.File.Insert(data, q1)
		if l != int64(len(data)) {
			return fmt.Errorf("Wrong number of inserted characters!")
		}
	}
	return nil
}

func cCmd(context innerContext, cmd Cmd) error {
	q0, q1 := context.File.Dot()
	return replaceText(context, q0, q1, []byte(cmd.text))
}

func dCmd(context innerContext, cmd Cmd) error {
	q0, q1 := context.File.Dot()
	if q1 > q0 {
		context.File.Delete(q0, q1)
	}
	return nil
}

func gCmd(context innerContext, cmd Cmd) error {
	re, err := compileRegexp(cmd.re)
	if err != nil {
		return err
	}
	q0, q1 := context.File.Dot()
	reader := context.File.Reader(q0, q1)
	location := re.FindReaderIndex(bufio.NewReader(reader))
	hasMatch := location != nil
	isInverse := cmd.cmdc == uint16('v')
	if (hasMatch && (!isInverse)) || ((!hasMatch) && isInverse) {
		context.File.Select(q0, q1)
		err := cmdExec(*cmd.cmd, context)
		if err != nil {
			return err
		}
	}
	return nil
}

func iCmd(context innerContext, cmd Cmd) error {
	q0, _ := context.File.Dot()
	l := int64(0)
	if len(cmd.text) > 0 {
		data := []byte(cmd.text)
		l = context.File.Insert(data, q0)
		if l != int64(len(data)) {
			return fmt.Errorf("Wrong number of inserted characters!")
		}
	}
	return nil
}

func mCmd(context innerContext, cmd Cmd) error {
	addr2, err := cmdAddress(cmd.mtaddr, context, 0)
	if err != nil {
		return err
	}
	q0, q1 := context.File.Dot()
	if q1 <= q0 {
		return nil
	}
	if q0 == addr2[0] && q1 == addr2[1] {
		return nil
	}
	if q1 <= addr2[0] || q0 >= addr2[1] {
		reader := context.File.Reader(q0, q1)
		data := make([]byte, q1-q0)
		_, err := io.ReadFull(reader, data)
		if err != nil {
			return err
		}
		context.File.Delete(q0, q1)
		context.File.Insert(data, addr2[1])
	} else {
		return fmt.Errorf("Move overlaps itself!")
	}
	return nil
}

func sCmd(context innerContext, cmd Cmd) error {
	re, err := compileRegexp(cmd.re)
	if err != nil {
		return err
	}
	rangesets := make([][]textRange, 0)
	q0, q1 := context.File.Dot()
	op := int64(-1)
	n := cmd.num
	reader := context.File.Reader(q0, q1)
	for p1 := q0; p1 <= q1; {
		_, err = reader.Seek(p1-q0, io.SeekStart)
		if err != nil {
			return err
		}
		location := re.FindReaderSubmatchIndex(bufio.NewReader(reader))
		if location == nil || len(location) < 2 {
			break
		}
		rangeset := make([]textRange, 0)
		for i := 0; i < len(location)/2; i += 1 {
			rangeset = append(rangeset, textRange{
				q0: int64(location[i*2]) + p1,
				q1: int64(location[i*2+1]) + p1,
			})
		}
		if rangeset[0].q0 == rangeset[0].q1 {
			if rangeset[0].q0 == op {
				p1 += 1
				continue
			}
			p1 = rangeset[0].q1 + 1
		} else {
			p1 = rangeset[0].q1
		}
		op = rangeset[0].q1
		n -= 1
		if n > 0 {
			continue
		}
		rangesets = append(rangesets, rangeset)
	}
	for _, rangeset := range rangesets {
		buf := make([]rune, 0)
		text := []rune(cmd.text)
		for i := 0; i < len(text); i++ {
			if text[i] == '\\' && i < len(text)-1 {
				i += 1
				ch := text[i]
				if ch >= '1' && ch <= '9' {
					j := int(ch - '0')
					if j >= len(rangeset) {
						return fmt.Errorf("Invalid replacement offset!")
					}
					_, err = reader.Seek(rangeset[j].q0-q0, io.SeekStart)
					if err != nil {
						return err
					}
					data := make([]byte, rangeset[j].q1-rangeset[j].q0)
					_, err = io.ReadAtLeast(reader, data, len(data))
					if err != nil {
						return nil
					}
					buf = append(buf, []rune(string(data))...)
				} else {
					buf = append(buf, ch)
				}
			} else if text[i] != '&' {
				buf = append(buf, text[i])
			} else {
				_, err = reader.Seek(rangeset[0].q0-q0, io.SeekStart)
				if err != nil {
					return err
				}
				data := make([]byte, rangeset[0].q1-rangeset[0].q0)
				_, err = io.ReadAtLeast(reader, data, len(data))
				if err != nil {
					return nil
				}
				buf = append(buf, []rune(string(data))...)
			}
		}
		err = replaceText(context, rangeset[0].q0, rangeset[0].q1, []byte(string(buf)))
		if err != nil {
			return err
		}
	}
	return nil
}

func tCmd(context innerContext, cmd Cmd) error {
	addr2, err := cmdAddress(cmd.mtaddr, context, 0)
	if err != nil {
		return err
	}
	q0, q1 := context.File.Dot()
	if q1 <= q0 {
		return nil
	}
	reader := context.File.Reader(q0, q1)
	data := make([]byte, q1-q0)
	_, err = io.ReadFull(reader, data)
	if err != nil {
		return err
	}
	context.File.Insert(data, addr2[1])
	return nil
}

func pCmd(context innerContext, cmd Cmd) error {
	q0, q1 := context.File.Dot()
	if context.Printer != nil {
		reader := context.File.Reader(q0, q1)
		_, err := io.Copy(context.Printer, reader)
		if err != nil {
			return err
		}
	}
	context.File.Select(q0, q1)
	return nil
}

func xCmd(context innerContext, cmd Cmd) error {
	if cmd.re != "" {
		return looper(context, cmd, cmd.cmdc == uint16('x'))
	} else {
		return lineLooper(context, cmd)
	}
}

func eqCmd(context innerContext, cmd Cmd) error {
	var mode int
	switch len(cmd.text) {
	case 0:
		mode = posnLine
	case 1:
		if cmd.text[0] == '#' {
			mode = posnChars
			break
		}
		if cmd.text[0] == '+' {
			mode = posnLineChars
			break
		}
		fallthrough
	default:
		return fmt.Errorf("Newline expected!")
	}
	return printPosn(context, mode)
}

func looper(context innerContext, cmd Cmd, isX bool) error {
	re, err := compileRegexp(cmd.re)
	if err != nil {
		return err
	}
	ranges := make([]textRange, 0)
	q0, q1 := context.File.Dot()
	op := q0
	if isX {
		op = -1
	}
	reader := context.File.Reader(q0, q1)
	for p := q0; p < q1; {
		var tr textRange
		_, err = reader.Seek(p-q0, io.SeekStart)
		if err != nil {
			return err
		}
		location := re.FindReaderIndex(bufio.NewReader(reader))
		if location == nil {
			if isX || op > q1 {
				break
			}
			tr.q0, tr.q1 = op, q1
			p = q1 + 1
		} else {
			l0 := int64(location[0]) + p
			l1 := int64(location[1]) + p
			if l0 == l1 {
				if l0 == op {
					p += 1
					continue
				}
				p = l1 + 1
			} else {
				p = l1
			}
			if isX {
				tr.q0, tr.q1 = l0, l1
			} else {
				tr.q0, tr.q1 = op, l0
			}
			op = l1
		}
		ranges = append(ranges, tr)
	}
	return loopCmd(context, *cmd.cmd, ranges)
}

func lineLooper(context innerContext, cmd Cmd) error {
	q0, q1 := context.File.Dot()
	a3 := textRange{
		q0: q0,
		q1: q0,
	}
	lineAddr, err := extractLineAddress(context, 0, 1, []int64{a3.q0, a3.q1})
	if err != nil {
		return err
	}
	ranges := make([]textRange, 0)
	for p := q0; p < q1; p = a3.q1 {
		a3.q0 = a3.q1
		if p != q0 || lineAddr[1] == p {
			lineAddr, err = extractLineAddress(context, 1, 1, []int64{a3.q0, a3.q1})
			if err != nil {
				return err
			}
		}
		if lineAddr[0] >= q1 {
			break
		}
		if lineAddr[1] >= q1 {
			lineAddr[1] = q1
		}
		if lineAddr[1] > lineAddr[0] {
			if lineAddr[0] >= a3.q1 && lineAddr[1] > a3.q1 {
				tr := textRange{
					q0: lineAddr[0],
					q1: lineAddr[1],
				}
				a3 = tr
				ranges = append(ranges, tr)
				continue
			}
		}
		break
	}
	return loopCmd(context, *cmd.cmd, ranges)
}

func loopCmd(context innerContext, cmd Cmd, ranges []textRange) error {
	for _, r := range ranges {
		context.File.Select(r.q0, r.q1)
		err := cmdExec(cmd, context)
		if err != nil {
			return err
		}
	}
	return nil
}

func regexpSearch(reStr string, context innerContext, start int64, end int64, sign int) ([]int64, error) {
	re, err := compileRegexp(reStr)
	if err != nil {
		return nil, err
	}
	reader := bufio.NewReader(context.File.Reader(start, end))
	location := re.FindReaderIndex(reader)
	if location == nil {
		return nil, nil
	}
	if sign < 0 {
		// Iterator more searches to look for last match
		for {
			start = int64(location[1]) + 1
			reader = bufio.NewReader(context.File.Reader(start, end))
			nextLocation := re.FindReaderIndex(reader)
			if nextLocation == nil {
				break
			}
			location = nextLocation
		}
	}
	return []int64{int64(location[0]) + start, int64(location[1]) + start}, nil
}

func extractLineAddress(context innerContext, lineNumber int64, sign int, currentAddress []int64) ([]int64, error) {
	q0, q1 := currentAddress[0], currentAddress[1]
	fileLen := context.File.Len()
	result := []int64{0, 0}
	if sign >= 0 {
		var p int64
		if lineNumber == 0 {
			if sign == 0 || q1 == 0 {
				return result, nil
			}
			result[0] = q1
			p = q1 - 1
		} else {
			var n uint64
			var reader io.Reader
			if sign == 0 || q1 == 0 {
				p = 0
				n = 1
				reader = context.File.Reader(p, fileLen)
			} else {
				p = q1 - 1
				n = 0
				reader = context.File.Reader(p, fileLen)
				data := []byte{0}
				_, err := reader.Read(data)
				if err != nil && err != io.EOF {
					return nil, err
				}
				p += 1
				if err != io.EOF && data[0] == '\n' {
					n = 1
				}
			}
			for n < uint64(lineNumber) {
				data := []byte{0}
				_, err := reader.Read(data)
				if err == io.EOF {
					return nil, fmt.Errorf("Address out of range")
				}
				if err != nil {
					return nil, err
				}
				p += 1
				if data[0] == '\n' {
					n += 1
				}
			}
			result[0] = p
		}
		reader := context.File.Reader(p, fileLen)
		for {
			data := []byte{0}
			_, err := reader.Read(data)
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			p += 1
			if data[0] == '\n' {
				break
			}
		}
		result[1] = p
	} else {
		p := q0
		reader := context.File.Reader(0, p)
		if lineNumber == 0 {
			result[1] = q0
		} else {
			for n := int64(0); n < lineNumber; {
				if p == 0 {
					n += 1
					if n != lineNumber {
						return nil, fmt.Errorf("Address out of range")
					}
				} else {
					_, err := reader.Seek(p-1, io.SeekStart)
					if err != nil {
						return nil, err
					}
					data := []byte{0}
					_, err = reader.Read(data)
					if err != nil {
						return nil, err
					}
					if data[0] != '\n' {
						p -= 1
					} else {
						n += 1
						if n != lineNumber {
							p -= 1
						}
					}
				}
			}
			result[1] = p
			if p > 0 {
				p -= 1
			}
		}
		for p > 0 {
			_, err := reader.Seek(p-1, io.SeekStart)
			if err != nil {
				return nil, err
			}
			data := []byte{0}
			_, err = reader.Read(data)
			if err != nil {
				return nil, err
			}
			if data[0] == '\n' {
				break
			}
			p -= 1
		}
		result[0] = p
	}
	return result, nil
}

func compileRegexp(reStr string) (*regexp.Regexp, error) {
	return regexp.Compile(fmt.Sprintf("(?m)%s", reStr))
}

func replaceText(context innerContext, q0 int64, q1 int64, data []byte) error {
	if q1 > q0 {
		context.File.Delete(q0, q1)
	}
	l := int64(0)
	if len(data) > 0 {
		l = context.File.Insert(data, q0)
		if l != int64(len(data)) {
			return fmt.Errorf("Wrong number of inserted characters!")
		}
	}
	return nil
}

func printPosn(context innerContext, mode int) error {
	var text string
	q0, q1 := context.File.Dot()
	switch mode {
	case posnChars:
		var secondText string
		if q1 != q0 {
			secondText = fmt.Sprintf(",#%d", q1)
		}
		text = fmt.Sprintf("#%d%s\n", q0, secondText)
	case posnLine:
		l1, _, err := lineEndingCount(context, 0, q0)
		if err != nil {
			return err
		}
		l1 += 1
		l2, _, err := lineEndingCount(context, q0, q1)
		if err != nil {
			return err
		}
		l2 += l1
		if q1 > 0 && q1 > q0 {
			reader := context.File.Reader(q1-1, q1)
			data := []byte{0}
			_, err := reader.Read(data)
			if err != nil {
				return err
			}
			if data[0] == '\n' {
				l2 -= 1
			}
		}
		var secondText string
		if l2 != l1 {
			secondText = fmt.Sprintf(",%d", l2)
		}
		text = fmt.Sprintf("%d%s\n", l1, secondText)
	case posnLineChars:
		l1, r1, err := lineEndingCount(context, 0, q0)
		if err != nil {
			return err
		}
		l1 += 1
		l2, r2, err := lineEndingCount(context, q0, q1)
		if err != nil {
			return err
		}
		l2 += l1
		var secondText string
		if l2 != l1 {
			secondText = fmt.Sprintf(",%d+#%d", l2, r2)
		}
		text = fmt.Sprintf("%d+#%d%s\n", l1, r1, secondText)
	}
	if context.Printer != nil {
		_, err := io.WriteString(context.Printer, text)
		if err != nil {
			return err
		}
	}
	return nil
}

func lineEndingCount(context innerContext, q0, q1 int64) (int64, int64, error) {
	nl := int64(0)
	start := q0
	reader := context.File.Reader(q0, q1)
	for q0 < q1 {
		data := []byte{0}
		_, err := reader.Read(data)
		if err != nil {
			return 0, 0, err
		}
		if data[0] == '\n' {
			start = q0 + 1
			nl += 1
		}
		q0 += 1
	}
	return nl, q0 - start, nil
}
