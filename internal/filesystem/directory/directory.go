package directory

import (
	"file-system/internal/filesystem/directory/record"
	"os"
)

type Directory struct {
	Records []record.Record
}

func ParseDirectoryFromBlock() Directory {
	// parses some block of data which should represent directory
	// returns Directory struct with all records in it
	return Directory{}
}

func (d Directory) WriteToFile(file *os.File, offset int) {
	// writes all records in directory to file at specified offset
}
