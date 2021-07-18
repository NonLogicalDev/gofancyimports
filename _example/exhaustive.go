// (before 1) [package doc comment]
package main

// (before 2) [this is a general comment]

// (group 1) this is a lonely single element grouping.
import (
	"flag/parser"
	"fmt"
	"github.com/sanity-io/litter"
	gtoken "go/token"
	"net"
	"net/http/httptrace"
	"net/mail"
	"os"
)

// (middle 1) [this comment says something about imports that follow?] [should pull up to the top]

import (
	"flag"
)

// (middle 2) [this comment says something about imports that follow?] [should pull up to the top]

// Test
import "image" // this is comment for 'image'

// (group 2) [this is a comment describing import group] [this should cause the group to stay]
/*
	 (group 2) multiline p1
*/ /*
 (group 2) multiline p2
*/
import (
	"net/mail"

	"net"

	"net/http/httptrace"
)

// (middle 3) [this comment says something about imports?] [should pull up to the top]

import (
	"go/parser"
	gtoken "go/token" // this comment is just for 'go/token'
)

// (group 3) [this is a comment describing import group] [this should cause the group to stay]
import (
	"a"

	/*
	   (group 3 mid) multiline p1
	*/

	"b"

	/*
	   (group 3 mid) multiline p2
	*/

	"c"

	/*
		(mid-import 2) [what does this even mean?]
	*/
)

import (
	// This comment is for group starting with 'os' (the group should be preserved as unit, but may be sorted)
	"os"
	"fmt"

	"github.com/sanity-io/litter"
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
		_ = fmt.Println
		_ = httptrace.DNSDoneInfo{}
		_ = flag.Bool
		_ = image.Rect
		_ = litter.Dump
		_ = mail.Address{}
	)
}

// Hello This is comment (after 3)
