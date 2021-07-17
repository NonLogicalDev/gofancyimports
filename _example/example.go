package main

import (
	"flag"
)

import "image" // this is comment for 'image'

// This is a comment for the second import group. (this should cause the group to stay)
import (
	"net/mail"

	"net"

	"net/http/httptrace"
)

import (
	"go/parser"
	gtoken "go/token" // this comment is just for 'go/token'

	// This comment is for group starting with 'os' (the group should be preserved as unit, but may be sorted)
	"os"
	"fmt"

	"github.com/sanity-io/litter"
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
