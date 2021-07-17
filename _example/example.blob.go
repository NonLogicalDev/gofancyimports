package main

import (
	"flag"
	"image" // this is comment for 'image'
	"net/mail"
	"net"
	"net/http/httptrace"
	"go/parser"
	gtoken "go/token" // this comment is just for 'go/token'
	"fmt"
	"os"
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
