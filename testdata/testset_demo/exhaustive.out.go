//go:build ignore

// (before 1) [package doc comment]
package main

// (before 2) [this is a general comment]

// (middle 1) [this comment says something about imports that follow?] [should pull up to the top]
// (middle 2) [this comment says something about imports that follow?] [should pull up to the top]
// (middle 3) [this comment says something about imports?] [should pull up to the top]
import (
	"flag"
	_ "flag"
	"go/parser"
	gtoken "go/token" // this comment is just for 'go/token'
	"sync"

	"github.com/sanity-io/litter"

	// This comment is for group starting with 'os' (fmt/os)
	// (the group should be preserved as unit, but may be sorted) (fmt/os)
	"fmt"
	"os"
)

// (group 1) this is a lonely single element grouping.
import "go/format"

// Test
import "image" // this is comment for 'image'

// (group 2) [this is a comment describing import group] [this should cause the group to stay]
/*
 (group 2) multiline p1
*/
/*
 (group 2) multiline p2
*/
import (
	"net"
	"net/http/httptrace"
	"net/mail"
)

// (group 3) [this is a comment describing import group] [this should cause the group to stay]
/*
   (group 3 mid) multiline p1
*/
/*
   (group 3 mid) multiline p2
*/
/*
	(mid-import 2) [what does this even mean?]
*/
import (
	"strconv"
	"strings"

	// bytes group
	"bytes"
)

// Hello This is comment (after 1)

/*
	test
*/
/*
	test2
*/
func main() {
	var (
		_ = net.UDPAddr{}
		_ = gtoken.FileSet{}
		_ = os.Args
		_ = parser.ParseFile
		_ = format.Node
		_ = fmt.Println
		_ = httptrace.DNSDoneInfo{}
		_ = flag.Bool
		_ = image.Rect
		_ = litter.Dump
		_ = mail.Address{}
		_ = bytes.NewReader
		_ = strings.NewReader
		_ = strconv.Itoa
	)
}

// Hello This is comment (after 3)
