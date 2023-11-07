package directory

import (
	"file-system/internal/filesystem/directory/record"
	"os"
)

type Directory struct {
	Records []record.Record
}

func CreateNewDirectory(inode uint32, parentInode uint32) Directory {
	currDir := record.NewRecord(inode, ".")
	parentDir := record.NewRecord(parentInode, "..")

	return Directory{
		Records: []record.Record{currDir, parentDir},
	}
}

func ParseDirectoryFromBlock() Directory {
	// parses some block of data which should represent directory
	// returns Directory struct with all records in it
	return Directory{}
}

func (d *Directory) AddFile(inode uint32, name string) {
	d.Records = append(d.Records, record.NewRecord(inode, name))
}

func (d Directory) WriteAt(file *os.File, offset uint32) error {
	for _, rec := range d.Records {
		err := rec.WriteAt(file, offset)
		if err != nil {
			return err
		}
		offset += uint32(rec.RecordLength)
	}
	return nil
}
