package imapclient

import (
	"bufio"
	"crypto/tls"
	"net/mail"
	//"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"
)

var _ = log.Print

type Client struct {
	conn *tls.Conn

	tagCnt uint16 // unused
}

type ListItem struct {
	Attrs []string
	Delim string
	Name  string
}

const (
	tagPrefix = 'A'

	FlagSeen     = "\\Seen"
	FlagAnswered = "\\Answered"
	FlagFlagged  = "\\Flagged"
	FlagDeleted  = "\\Deleted"
	FlagDraft    = "\\Draft"
	FlagRecent   = "\\Recent"
)

func NewClient(network, addr string) (*Client, error) {
	conn, err := tls.Dial(network, addr, &tls.Config{ServerName: strings.Split(addr, ":")[0]})
	if err != nil {
		return nil, err
	}

	//consume all response
	buff := make([]byte, 255)
	for {
		n, err := conn.Read(buff)
		if err != nil {
			return nil, err
		}
		if n != len(buff) {
			break
		}
	}

	return &Client{
		conn:   conn,
		tagCnt: 0,
	}, nil
}

func (c *Client) LeakTLSConn() *tls.Conn {
	return c.conn
}

func (c *Client) Noop() error {
	_, err := c.Command("NOOP")
	return err
}

func (c *Client) Capability() (capabilities []string, err error) {
	res, err := c.Command("CAPABILITY")
	if err != nil {
		return nil, err
	}
	return strings.Split(res, " "), nil
}

func (c *Client) StartTLS() error {
	return fmt.Errorf("not implemented")
}

func (c *Client) Authenticate(mechaname string) ([]byte, error) {
	return nil, fmt.Errorf("not implemented")
}

func (c *Client) Login(username, password string) error {
	_, err := c.Command(fmt.Sprintf("LOGIN %v %v", username, password))
	return err
}

func (c *Client) Select(mailbox string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("SELECT %v", mailbox))
	return err
}

func (c *Client) Examine(mailbox string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("EXAMINE %v", mailbox))
	return err
}

func (c *Client) Create(mailbox string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("CREATE %v", mailbox))
	return err
}

func (c *Client) Delete(mailbox string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("DELETE %v", mailbox))
	return err
}

func (c *Client) Rename(mailbox, newname string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("RENAME %v %v", mailbox, newname))
	return err
}

func (c *Client) Subscribe(mailbox string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("SUBSCRIBE %v", mailbox))
	return err
}

func (c *Client) Unsubscribe(mailbox string) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}
	_, err = c.Command(fmt.Sprintf("UNSUBSCRIBE %v", mailbox))
	return err
}

func (c *Client) List(reference, mailbox string) ([]ListItem, error) {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return nil, fmt.Errorf("failed to encode mailbox: %v", err)
	}
	res, err := c.Command(fmt.Sprintf("LIST \"%v\" \"%v\"", reference, mailbox))
	if err != nil {
		return nil, err
	}

	items := make([]ListItem, 0, 10)

	s := bufio.NewScanner(strings.NewReader(res))
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "* LIST") {
			break
		}

		// attrs
		posAttrSt := strings.Index(line, "(")
		posAttrEd := strings.Index(line, ")")
		// delim
		posDlmSt := strings.Index(line[posAttrEd+1:], "\"") + posAttrEd + 1
		posDlmEd := strings.Index(line[posDlmSt+1:], "\"") + posDlmSt + 1
		// name
		posNmSt := strings.Index(line[posDlmEd+1:], "\"") + posDlmEd + 1
		posNmEd := strings.Index(line[posNmSt+1:], "\"") + posNmSt + 1

		name8, err := DecodeModifiedUTF7([]byte(line[posNmSt+1 : posNmEd]))
		if err != nil {
			name8 = []byte(line[posNmSt+1 : posNmEd])
		}
		items = append(items, ListItem{
			Attrs: strings.Split(line[posAttrSt+1:posAttrEd], " "),
			Delim: line[posDlmSt+1 : posDlmEd],
			Name:  string(name8),
		})
	}
	return items, nil
}

func (c *Client) LSub(reference, mailbox string) /* hatena    []ListItem,*/ error {
	return fmt.Errorf("not implemented")
}

func (c *Client) Status(mailbox string, itemNames []string) (map[string]uint32, error) {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return nil, fmt.Errorf("failed to encode mailbox: %v", err)
	}
	res, err := c.Command(fmt.Sprintf("STATUS \"%v\" (%v)", mailbox, strings.Join(itemNames, " ")))
	if err != nil {
		return nil, err
	}

	s := bufio.NewScanner(strings.NewReader(res))
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "* STATUS") {
			break
		}

		posPE := strings.LastIndex(line, ")")
		if posPE == -1 {
			return nil, nil
		}
		posPS := strings.LastIndex(line, "(")
		if posPE == -1 {
			return nil, fmt.Errorf("failed to parse status")
		}

		st := line[posPS+1 : posPE]
		sts := strings.Split(st, " ")
		if len(sts)%2 == 1 {
			return nil, fmt.Errorf("not paired (last:%v)", sts[len(sts)-1])
		}

		m := make(map[string]uint32)
		for i := 0; i < len(sts)/2; i++ {
			v, err := strconv.ParseUint(sts[i+1], 10, 32)
			if err != nil {
				return nil, fmt.Errorf("unexpected value of %v %v", sts[i], sts[i+1])
			}
			m[sts[i]] = uint32(v)
		}
		return m, nil
	}

	return nil, nil
}

func (c *Client) Append(mailbox string, flags []string, message mail.Message) error {
	mailbox, err := EncodeModifiedUTF7String(mailbox)
	if err != nil {
		return fmt.Errorf("failed to encode mailbox: %v", err)
	}

	addIfMissing(message.Header, "Content-Type", "text/plain; charset=\"utf-8\"")
	addIfMissing(message.Header, "MIME-Version", "1.0")
	addIfMissing(message.Header, "Content-Transfer-Encoding", "base64")
	addIfMissing(message.Header, "Date", time.Now().Format(time.RFC1123Z))

	body, err := ioutil.ReadAll(message.Body)
	if err != nil {
		return fmt.Errorf("failed to read body from message: %v", err)
	}

	contentLines := make([]string, 0, 10)
	for k, v := range message.Header {
		contentLines = append(contentLines, k+": "+strings.Join(v, ""))
	}
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, string(body))
	contentLines = append(contentLines, "")
	contentLines = append(contentLines, "")
	contents := strings.Join(contentLines, "\r\n")
	contentLength := len(contents)

	var flagPart string
	if len(flags) == 0 {
		flagPart = ""
	} else {
		flagPart = "(" + strings.Join(flags, " ") + ") "
	}

	//log.Println("==================================================")
	//log.Printf(os.Stderr, "%v\n", contents)
	//log.Println("==================================================")

	_, err = c.Command(fmt.Sprintf("APPEND \"%v\" %v{%v}", mailbox, flagPart, contentLength))
	if err != nil {
		//log.Printf("err: %v\n", err)
		return err
	}

	//log.Println("sending contents")
	_, err = c.Raw("", contents+"\r\n")
	if err != nil {
		//log.Printf("err: %v\n", err)
		return err
	}
	//_, err = c.Command(contents)
	//if err != nil {
	//	return err
	//}

	return nil
}

func (c *Client) Search(criteria string, optLiteral ...string) ([]uint32, error) {
	if criteria == "" {
		criteria = "ALL"
	}

	var res string
	var err error
	if len(optLiteral) == 0 {
		res, err = c.Command("SEARCH " + criteria)
		if err != nil {
			return nil, err
		}
	} else {
		res, err = c.Command(fmt.Sprintf("SEARCH CHARSET UTF-8 %s {%d}", criteria, len(optLiteral[0])))
		if err != nil {
			return nil, err
		}
		res, err = c.Raw("", optLiteral[0]+"\r\n")
		if err != nil {
			return nil, err
		}
	}

	s := bufio.NewScanner(strings.NewReader(res))
	for s.Scan() {
		line := s.Text()
		if !strings.HasPrefix(line, "* SEARCH") {
			break
		}

		//log.Printf("line:%q\n", line)
		idStrs := strings.Split(strings.Trim(line[8:], " "), " ")
		//log.Printf("idStrs:%#v\n", idStrs)
		ids := make([]uint32, 0, len(idStrs))
		for _, id := range idStrs {
			if id == "" {
				break
			}
			v, err := strconv.ParseUint(id, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("unexpected id %v", id)
			}
			ids = append(ids, uint32(v))
		}
		return ids, nil
	}

	return nil, nil
}

func (c *Client) Fetch(seqSet string) (map[uint32]*mail.Message, error) {
	res, err := c.Command(fmt.Sprintf("FETCH %v (BODY.PEEK[])", seqSet))
	if err != nil {
		return nil, err
	}

	mails := make(map[uint32]*mail.Message)

	s := bufio.NewScanner(strings.NewReader(res))
	for s.Scan() {
		line := s.Text()

		// beginning of one of messages
		if strings.HasPrefix(line, "*") {
			if strings.Index(line, "FETCH") == -1 {
				continue
			}

			// parse sequence
			posSP1 := strings.Index(line, " ")
			posSP2 := posSP1 + 1 + strings.Index(line[posSP1+1:], " ")
			seqStr := line[posSP1+1 : posSP2]
			seq64, err := strconv.ParseUint(seqStr, 10, 32)
			if err != nil {
				return nil, fmt.Errorf("unexpected seq %v (line=%v, sp1=%v, sp2=%v): %v", seqStr, line, posSP1, posSP2, err)
			}
			seq := uint32(seq64)

			// parse meessage
			rawmsg := make([]string, 0, 10)
			for s.Scan() {
				line := s.Text()
				if strings.HasPrefix(line, ")") {
					// end of parsing
					r := strings.NewReader(strings.Join(rawmsg, "\r\n"))
					m, err := mail.ReadMessage(r)
					if err != nil {
						return nil, fmt.Errorf("failed to read message (of seq %v): %v", seq, err)
					}

					mails[seq] = m
					break // end parsing message
				}
				rawmsg = append(rawmsg, line)
			}
		}
	}

	return mails, nil
}

func (c *Client) Store(seqSet, dataItem string, flags []string) error {
	_, err := c.Command(fmt.Sprintf("STORE %v %v (%s)", seqSet, dataItem, strings.Join(flags, " ")))
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Expunge() error {
	_, err := c.Command("EXPUNGE")
	if err != nil {
		return err
	}
	return nil
}

func (c *Client) Logout() error {
	_, err := c.Command("LOGOUT")
	return err
}

func (c *Client) Raw(tag, raw string) (string, error) {
	if _, err := c.conn.Write([]byte(raw)); err != nil {
		return "", err
	}

	//log.Println("==========================================")
	//log.Print(raw)
	//log.Println("------------------------------------------")

	// receive response and parse it
	var resSt string
	var resLastMyMsg string
	var resMsg string
	s := bufio.NewScanner(c.conn)
	//log.Printf("[%v] scanning\n", tag)
	for s.Scan() {
		//log.Printf("[%v] s.text()\n", tag)
		resline := s.Text()
		//log.Printf("[%v] %v\n", tag, resline)

		if len(resline) > 0 && resline[0] == '+' {
			//log.Printf("%v> %v\n", tag, resline)
			resSt = "+"
			resLastMyMsg = resline
			break
		}

		//log.Printf("%s> %v\n", tag, resline)

		resMsg += resline + "\r\n"

		if len(resline) == 0 {
			continue
		}

		if resline[0] == tagPrefix {
			if resSt == "" {
				stcomps := strings.Split(resline, " ")
				//log.Printf("status components: %#v\n", stcomps)
				var st string
				if len(stcomps) >= 2 {
					//TAG RESST REMAININGS
					st = stcomps[1]
				}
				//log.Printf("status: %v\n", st)

				switch st {
				case "OK":
				case "NO":
				case "BAD":
				case "PREAUTH":
				case "BYE":
					//NO fallthrough
				default:
					st = ""
				}
				resSt = st
			}
		}

		if resSt != "" {
			resLastMyMsg = resline
			break
		}
	}
	if err := s.Err(); err != nil {
		return "", fmt.Errorf("failed to scan result: ", err)
	}

	//log.Printf("resSt:%v, resLastMyMsg:%v", resSt, string(resLastMyMsg))
	if resSt != "OK" && resSt != "+" {
		//log.Printf("not OK nor +: %v\n", string(resLastMyMsg))
		return resMsg, fmt.Errorf("%v", string(resLastMyMsg))
	}
	return resMsg, nil
}

func (c *Client) Command(cmd string) (string, error) {
	tag := c.makeNewTag()
	raw := fmt.Sprintf("%v %v\r\n", tag, cmd)

	return c.Raw(tag, raw)
}

func (c *Client) makeNewTag() string {
	c.tagCnt = (c.tagCnt + 1) % 1000
	return fmt.Sprintf("%c%d", tagPrefix, c.tagCnt)
}

func addIfMissing(m mail.Header, key, value string) {
	if _, found := m[key]; !found {
		m[key] = []string{value}
	}
}
