package imapclient

import (
	"bytes"
	"encoding/base64"
	"fmt"
	//"log"

	"golang.org/x/text/encoding/unicode"
)

func EncodeModifiedUTF7String(src string) (dst string, err error) {
	dstbytes, err := EncodeModifiedUTF7([]byte(src))
	if err != nil {
		return "", err
	}
	return string(dstbytes), nil
}

func DecodeModifiedUTF7String(src string) (dst string, err error) {
	dstbytes, err := DecodeModifiedUTF7([]byte(src))
	if err != nil {
		return "", err
	}
	return string(dstbytes), nil
}

func DecodeModifiedUTF7(src []byte) (dst []byte, err error) {
	dst = nil

	var posAmp, posHypen int

	for {
		posAmp = bytes.IndexByte(src, '&')
		if posAmp == -1 {
			dst = append(dst, src...)
			break
		} else {
			// before &
			dst = append(dst, src[:posAmp]...)

			posHypen = bytes.IndexByte(src[posAmp:], '-')
			if posHypen == -1 {
				return nil, fmt.Errorf("- matching to & is missing")
			}

			if posAmp+1 == posHypen {
				dst = append(dst, '&')
			} else {
				// & x x x -
				//   x x x    : nonprintables
				nonprintables := src[posAmp+1 : posHypen]
				b64decoded, err := DecodeModifiedBase64(nonprintables)
				if err != nil {
					return nil, err
				}
				u16beEncoding := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
				u16beDecoder := u16beEncoding.NewDecoder()
				u16decoded, err := u16beDecoder.Bytes(b64decoded)
				if err != nil {
					return nil, err
				}
				dst = append(dst, u16decoded...)
			}
		}

		src = src[posHypen+1:]
	}

	return dst, nil
}

func DecodeModifiedBase64(src []byte) (dst []byte, err error) {
	src = bytes.Replace(src, []byte{','}, []byte{'/'}, -1)

	padding := 4 - len(src)&3
	if padding == 3 {
		return nil, fmt.Errorf("incorrect form")
	}

	src = append(src, bytes.Repeat([]byte{'='}, padding%4)...)
	dst = make([]byte, base64.StdEncoding.DecodedLen(len(src)))
	n, err := base64.StdEncoding.Decode(dst, src)
	if err != nil {
		return nil, err
	}
	dst = dst[:n]

	return dst, nil
}

func EncodeModifiedUTF7(src []byte) (dst []byte, err error) {
	dst = nil

	var posNext int

	for {
		posNext = len(src)

		// non-printable
		posNP := bytes.IndexFunc(src, isNonprintable)
		if posNP == -1 {
			dst = append(dst, bytes.Replace(src, []byte{'&'}, []byte{'&', '-'}, -1)...)
			break

		} else {
			// before non-printable
			dst = append(dst, bytes.Replace(src[:posNP], []byte{'&'}, []byte{'&', '-'}, -1)...)

			// printable
			posP := posNP + bytes.IndexFunc(src[posNP:], isPrintable)
			var nonprintables []byte
			//log.Printf("dst %v\n", dst)
			//log.Printf("NON-PRINTABLE %v\n", posNP)
			//log.Printf("PRINTABLE     %v\n", posP)
			//log.Printf("src %v\n", src)
			if posP == -1 {
				nonprintables = src[posNP:]
				posNext = len(src)
			} else {
				nonprintables = src[posNP:posP]
				posNext = posP
			}
			//log.Printf("nonprintables %v\n", nonprintables)

			u16beEncoding := unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM)
			u16beEncoder := u16beEncoding.NewEncoder()
			u16encoded, err := u16beEncoder.Bytes(nonprintables)
			if err != nil {
				return nil, err
			}
			b64encoded, err := EncodeModifiedBase64(u16encoded)
			if err != nil {
				return nil, err
			}

			dst = append(dst, '&')
			dst = append(dst, b64encoded...)
			dst = append(dst, '-')
		}

		src = src[posNext:]
	}

	return dst, nil
}

func isPrintable(r rune) bool {
	return 0x20 <= r && r <= 0x7e
}

func isNonprintable(r rune) bool {
	return r < 0x20 || 0x7e < r
}

func EncodeModifiedBase64(src []byte) (dst []byte, err error) {
	dst = make([]byte, base64.StdEncoding.EncodedLen(len(src)))
	base64.StdEncoding.Encode(dst, src)
	dst = bytes.Replace(dst, []byte{'/'}, []byte{','}, -1)
	posPad := bytes.Index(dst, []byte{'='})
	if posPad == -1 {
		return dst, nil
	}
	return dst[:posPad], nil
}
