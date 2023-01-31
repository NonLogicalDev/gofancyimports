// Package main
//
// Transforms:
//
// 	import (
// 		"github.com/sanity-io/litter"
// 		"flag"
// 	)
//
// 	import (
// 		_ "net/http/pprof"
// 		"os"
// 		"strconv"
// 		"gen/mocks/github.com/go-redis/redis"
// 		"github.com/go-redis/redis"
// 		"strings"
// 		"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
// 	)
//
// Into:
//
// 	import (
// 		"flag"
// 		"os"
// 		"strconv"
// 		"strings"
//
// 		"gen/mocks/github.com/go-redis/redis"
//
// 		"github.com/go-redis/redis"
// 		"github.com/sanity-io/litter"
//
// 		"github.com/NonLogicalDev/gofancyimports/internal/stdlib"
//
// 		_ "net/http/pprof"
// 	)
package main

