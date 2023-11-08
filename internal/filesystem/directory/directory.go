package directory

import (
	"encoding/binary"
	"file-system/internal/filesystem/directory/record"
	"fmt"
	"io"
	"os"
)

type Directory struct {
	Records []record.Record
}

func CreateNewDirectory(inode uint32, parentInode uint32) *Directory {
	currDir := record.NewRecord(inode, ".")
	parentDir := record.NewRecord(parentInode, "..")

	return &Directory{
		Records: []record.Record{currDir, parentDir},
	}
}

func ReadDirectoryAt(file *os.File, offset uint32) (*Directory, error) {
	directory := Directory{}
	for {
		inodeData := make([]byte, 4)
		_, err := file.ReadAt(inodeData, int64(offset))
		if err != nil {
			if err == io.EOF {
				break // End of directory
			}
			return &directory, err
		}
		inode := binary.BigEndian.Uint32(inodeData)
		offset += 4

		recordLengthData := make([]byte, 2)
		_, err = file.ReadAt(recordLengthData, int64(offset))
		if err != nil {
			return &directory, err
		}
		recordLength := binary.BigEndian.Uint16(recordLengthData)
		if recordLength == 0 {
			break // Empty record was read
		}
		offset += 2

		nameLengthData := make([]byte, 1)
		_, err = file.ReadAt(nameLengthData, int64(offset))
		if err != nil {
			return &directory, err
		}
		nameLength := nameLengthData[0]
		offset += 1

		nameData := make([]byte, nameLength)
		_, err = file.ReadAt(nameData, int64(offset))
		if err != nil {
			return &directory, err
		}
		name := string(nameData)
		offset += uint32(nameLength)

		record := record.Record{
			Inode:        inode,
			RecordLength: recordLength,
			NameLength:   nameLength,
			Name:         name,
		}
		directory.Records = append(directory.Records, record)
	}

	return &directory, nil
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

func (d Directory) ListRecords() {
	for _, record := range d.Records {
		fmt.Println(record.Name)
	}
}

func (d Directory) GetInode(recordName string) (uint32, error) {
	for _, r := range d.Records {
		if r.Name == recordName {
			return r.Inode, nil
		}
	}
	return 0, fmt.Errorf("record not found - %s", recordName)
}
