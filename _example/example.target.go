package main

import (
	"flag"
	"fmt"
	"image" // this is comment for 'image'
	"os"

	// This comment is for group starting with 'fmt' (the group should be preserved as unit, but may be sorted)
	"go/parser"
	gtoken "go/token" // this comment is just for 'go/token'

	"github.com/sanity-io/litter"
)

// This is a comment for the second import group. (this should cause the group to stay)
import (
	"net"
	"net/http/httptrace"
	"net/mail"
)

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
