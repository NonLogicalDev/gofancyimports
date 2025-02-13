package stdlib

import xstdlib "github.com/NonLogicalDev/gofancyimports/internal/stdlib/go_x_stdlib"

func IsStdlib(path string) bool {
	return xstdlib.HasPackage(path)
}
