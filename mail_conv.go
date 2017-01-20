package imapclient

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/text/encoding/japanese"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"

	"bitbucket.org/shu/log"
)

var _ = log.Print

func EncodeMailMessage(src *mail.Message) (dst *mail.Message, err error) {
	if src == nil {
		return nil, nil
	}

	encoding := "base64"

	buff := new(bytes.Buffer)

	// encode each header items and put it into buff
	for hk, hv := range src.Header {
		encoded := mime.BEncoding.Encode("UTF-8", hv[0])
		encoded = strings.Replace(encoded, "?b?", "?B?", -1)
		encoded = strings.Replace(encoded, "?q?", "?Q?", -1)
		if hk == "Content-Transfer-Encoding" {
			encoding = hv[0]
		}
		buff.WriteString(hk)
		buff.WriteString(": ")
		buff.WriteString(encoded)
		buff.WriteString("\r\n")
	}

	buff.WriteString("\r\n")

	// put body into buff
	// encode body according to Content-Transfer-Encoding header
	switch strings.ToLower(encoding) {
	case "base64":
		body, err := ioutil.ReadAll(src.Body)
		if err != nil {
			return nil, fmt.Errorf("body reading error: %v", err)
		}
		//log.Printf("BEFORE %v\n", string(body))
		n := base64.StdEncoding.EncodedLen(len(body))
		encodedBody := make([]byte, n)
		base64.StdEncoding.Encode(encodedBody, body)
		buff.Write(encodedBody)
		//log.Printf("AFTER  %v\n", string(encodedBody))
	}

	buff.WriteString("\r\n")
	//log.Printf("BUFF   =   %v\n", string(buff.Bytes()))

	// create mail.Message
	dst, err = mail.ReadMessage(buff)
	if err != nil {
		//log.Printf("buff=%v\n", string(buff.Bytes()))
		return nil, fmt.Errorf("message reading error: %v", err)
	}

	return dst, nil
}

// http://d.hatena.ne.jp/taknb2nch/20140212/1392198485
func DecodeMailMessage(src *mail.Message, optOnlyHeader ...bool) (dst []*mail.Message, err error) {
	if src == nil {
		return nil, nil
	}

	mediatype, params, err := mime.ParseMediaType(src.Header.Get("Content-Type"))
	if err != nil {
		return nil, fmt.Errorf("failed to parse media type: %v", err)
	}

	switch strings.Split(mediatype, "/")[0] {
	case "text", "html", "message":
		dst, err := decodeMailMessagePart(src, optOnlyHeader...)
		if err != nil {
			return nil, err
		}
		return []*mail.Message{dst}, err

	case "multipart":
		dsts := []*mail.Message{}

		boundary := params["boundary"]
		r := multipart.NewReader(src.Body, boundary)
		for {
			part, _ := r.NextPart()
			if part == nil {
				break
			}

			s2 := *src
			s2.Body = part
			for k, v := range part.Header {
				s2.Header[k] = v
			}

			dst, err := decodeMailMessagePart(&s2, optOnlyHeader...)
			if err != nil {
				return nil, fmt.Errorf("failed to parse part: %v", err)
			}
			dsts = append(dsts, dst)
		}
		return dsts, nil
	}

	//log.Debug("mail_conv:DecodeMailMessage: no matches")
	return nil, nil
}

func decodeMailMessagePart(src *mail.Message, optOnlyHeader ...bool) (dst *mail.Message, err error) {
	if src == nil {
		return nil, nil
	}

	encoding := ""

	/*
		charset := "UTF-8"
		if csPart := src.Header.Get("Content-Type"); csPart != "" {
			posCS := strings.Index(csPart, "charset=")
			posDelim := strings.Index(csPart[posCS+8:], ";")
			if posDelim == -1 {
				//log.Printf("posDelim=-1\n")
				posDelim = len(csPart)
			} else {
				posDelim += posCS + 8
			}
			//log.Printf("csPart=[%v]\n", csPart)
			//log.Printf("csPart[posCS+8:]=[%v]\n", csPart[posCS+8:])
			//log.Printf("posCS=[%v], posDelim=[%v]\n", posCS, posDelim)
			csQuoted := csPart[posCS+8 : posDelim]
			//log.Printf("csQuoted=[%v]\n", csQuoted)

			posStQuot := strings.Index(csQuoted, "\"")
			posEdQuot := strings.LastIndex(csQuoted, "\"")
			if posEdQuot == -1 {
				posStQuot = -1
				posEdQuot = len(csQuoted)
			}
			charset = csQuoted[posStQuot+1 : posEdQuot]
		}
	*/

	buff := new(bytes.Buffer)

	// decode each header items and put it into buff
	mimeDecoder := new(mime.WordDecoder)
	mimeDecoder.CharsetReader = func(charset string, input io.Reader) (io.Reader, error) {
		//e, err := ianaindex.MIME.Get(charset) //TODO panic
		switch strings.ToLower(charset) {
		case "iso-2022-jp":
			decoder := japanese.ISO2022JP.NewDecoder()
			return decoder.Reader(input), nil
		}
		//for _, enc := range japanese.All {
		//	name, _ := ianaindex.MIME.Name(enc)
		//	if strings.ToLower(charset) == strings.ToLower(name) {
		//		decoder := enc.NewDecoder()
		//		return decoder.Reader(input), nil
		//	}
		//}
		return nil, fmt.Errorf("unhandled charset %q", charset)
	}
	guessEncoding := ""
	for hk, hv := range src.Header {
		decoded, err := mimeDecoder.DecodeHeader(hv[0])
		if err != nil {
			return nil, fmt.Errorf("header decoding error %v: %v", hk, err)
		}
		if hk == "Content-Transfer-Encoding" {
			encoding = hv[0]
		} else if strings.Index(hv[0], "?q?") != -1 || strings.Index(hv[0], "?Q?") != -1 {
			guessEncoding = "quoted-printable"
		}
		buff.WriteString(hk)
		buff.WriteString(": ")
		buff.WriteString(decoded)
		buff.WriteString("\r\n")
	}

	//log.Printf("encoding=%v, guessEncoding=%v\n", encoding, guessEncoding)
	if encoding == "" && guessEncoding != "" {
		encoding = guessEncoding
	}
	buff.WriteString("\r\n")

	if len(optOnlyHeader) == 0 || !optOnlyHeader[0] {
		// put body into buff
		// decode body according to Content-Transfer-Encoding header
		switch strings.ToLower(encoding) {
		case "base64":
			body, err := ioutil.ReadAll(src.Body)
			if err != nil {
				return nil, fmt.Errorf("body reading error: %v", err)
			}
			//log.Printf("BEFORE %v\n", string(body))
			n := base64.StdEncoding.DecodedLen(len(body))
			decodedBody := make([]byte, n)
			n, err = base64.StdEncoding.Decode(decodedBody, body)
			if err != nil {
				return nil, fmt.Errorf("body decoding error: %v", err)
			}
			buff.Write(decodedBody[0:n])
			//log.Printf("AFTER  %v\n", string(encodedBody))

		case "quoted-printable":
			decReader := quotedprintable.NewReader(src.Body)
			decodedBody, err := ioutil.ReadAll(decReader)
			if err != nil {
				return nil, fmt.Errorf("body reading error q: %v", err)
			}
			buff.Write(decodedBody)

		case "":
			body, err := ioutil.ReadAll(src.Body)
			if err != nil {
				return nil, fmt.Errorf("body reading error: %v", err)
			}
			buff.Write(body)
		}
	}

	// create mail.Message
	dst, err = mail.ReadMessage(buff)
	if err != nil {
		log.Printf("DECODED\n%s\n", buff)
		return nil, fmt.Errorf("message reading error: %v", err)
	}

	return dst, nil
}
