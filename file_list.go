package main

import (
	"os"
)

type FileToScan struct {
	Path string
	Info os.FileInfo
	Err  error
}
