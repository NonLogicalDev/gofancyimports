package stdlib

//go:generate go run mkstdlib.go

func IsStdlib(path string) bool {
	_, ok := stdlib[path]
	return ok
}
