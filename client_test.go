package imapclient

import (
	"bytes"
	//"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/mail"
	"os"
	"strings"
	"testing"
)

/*
* env IMAP_USER
* enc IMAP_PASS
* */

func TestConnect(t *testing.T) {
	c, err := NewClient("tcp", "imap.gmail.com:993")
	if err != nil {
		t.Errorf("NewClient %v\n", err)
	}

	//A1
	response, err := c.Command("hoge")
	if err == nil || !strings.HasPrefix(string(response), "A1 BAD Unknown command") {
		t.Errorf("Command(hoge) err:%v res:%v\n", err, string(response))
	}

	//A2
	response, err = c.Command("NOOP")
	if err != nil || !strings.HasPrefix(response, "A2 OK") {
		t.Errorf("Command(NOOP) err:%v res:%v\n", err, string(response))
	}

	//A3
	response, err = c.Command("LOGIN " + os.Getenv("IMAP_USER") + " hoge")
	if err == nil || !strings.HasPrefix(response, "A3 NO") {
		t.Errorf("Command(LOGIN) err:%v res:%#v\n", err, string(response))
	}

	/*
		response, err = c.Command("LOGIN " + os.Getenv("IMAP_USER") + " " + os.Getenv("IMAP_PASS"))
		if err != nil || !strings.Contains(response, "A4 OK") {
			t.Errorf("Command(LOGIN2nd) err:%v res:%#v\n", err, string(response))
		}
	*/
	err = c.Login(os.Getenv("IMAP_USER"), os.Getenv("IMAP_PASS"))
	if err != nil {
		t.Errorf("Login err:%v\n", err)
	}

	items, err := c.List("", "*")
	items = items
	if err != nil {
		t.Errorf("List err:%v\n", err)
	} else {
		//for i, v := range items {
		//	fmt.Fprintf(os.Stderr, "Item %v %#v\n", i, v)
		//}
	}

	items, err = c.List("", "*pom*")
	if err != nil {
		t.Errorf("List err:%v\n", err)
	} else {
		//for i, v := range items {
		//	fmt.Fprintf(os.Stderr, "Item %v %#v\n", i, v)
		//}
	}

	st, err := c.Status("Notes/pomera_sync", []string{"MESSAGES"})
	st = st
	if err != nil {
		t.Errorf("List err:%v\n", err)
	} else {
		//for k, v := range st {
		//	fmt.Fprintf(os.Stderr, "Item %v %v\n", k, v)
		//}
	}

	err = c.Select("Notes/pomera_sync")
	if err != nil {
		t.Errorf("List err:%v\n", err)
	}

	ids, err := c.Search("")
	if err != nil {
		t.Errorf("List err:%v\n", err)
	} else if len(ids) != 2 {
		t.Errorf("Fetch len:%v\n", len(ids))
	} else {
		//for i, id := range ids {
		//	fmt.Fprintf(os.Stderr, "Item %v %v\n", i, id)
		//}
	}

	mm, err := c.Fetch("1:2,4")
	if err != nil {
		t.Errorf("Fetch err:%v\n", err)
	} else {
		for _, m := range mm {
			dm, err := DecodeMailMessage(m)
			if err != nil {
				t.Errorf("DecodeMailMessage error: %v", err)
			} else {
				fmt.Fprintf(os.Stderr, "header[%#v]\n", dm.Header)

				body, err := ioutil.ReadAll(dm.Body)
				if err != nil {
					t.Errorf("DecodeMailMessage error: %v", err)
				} else {
					fmt.Fprintf(os.Stderr, "body[%#v]\n", string(body))
				}
			}
		}
	}

	c.Noop()
	c.Logout()

	//A6

}
