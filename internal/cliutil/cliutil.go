package cliutil

import (
	"io"
	"os"
	"os/user"
)

func CurrentUsername() string {
	u, err := user.Current()
	if err != nil {
		return os.Getenv("USER")
	}
	return u.Username
}

func IsTTY(v any) bool {
	switch x := v.(type) {
	case *os.File:
		info, err := x.Stat()
		if err != nil {
			return false
		}
		return info.Mode()&os.ModeCharDevice != 0
	case io.Writer:
		return isTTYFromWriter(x)
	case io.Reader:
		return isTTYFromReader(x)
	default:
		return false
	}
}

func isTTYFromWriter(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return IsTTY(f)
}

func isTTYFromReader(r io.Reader) bool {
	f, ok := r.(*os.File)
	if !ok {
		return false
	}
	return IsTTY(f)
}
