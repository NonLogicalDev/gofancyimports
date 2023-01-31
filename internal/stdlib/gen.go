package stdlib

//go:generate go run mkstdlib.go

// IsStdlib is a helper function that can tell if a given package is a part of standard library.
//
// DEPRECATED: this is for reference only, it is possible for this package to fall behind the mainline go.
func IsStdlib(path string) bool {
	_, ok := stdlib[path]
	return ok
}
