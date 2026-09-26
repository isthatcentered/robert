package remove

import "os"

type Directory struct{}

func (Directory) Remove(path string) error { return os.RemoveAll(path) }
