package directory

import (
	"encoding/binary"
	"file-system/internal/filesystem/directory/record"
	"fmt"
	"io"
	"os"
)

type Directory struct {
	records map[string]record.Record
}

func CreateNewDirectory(inode uint32, parentInode uint32) *Directory {
	currDir := record.NewRecord(inode, ".")
	parentDir := record.NewRecord(parentInode, "..")

	records := make(map[string]record.Record)
	records[currDir.Name] = currDir
	records[parentDir.Name] = parentDir

	return &Directory{
		records: records,
	}
}

func ReadDirectoryAt(file *os.File, offset uint32) (*Directory, error) {
	directory := Directory{
		records: make(map[string]record.Record),
	}
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
		directory.records[record.Name] = record
	}

	return &directory, nil
}

func (d *Directory) AddFile(inode uint32, name string) {
	record := record.NewRecord(inode, name)
	d.records[record.Name] = record
}

func (d *Directory) DeleteFile(name string) {
	delete(d.records, name)
}

func (d Directory) WriteAt(file *os.File, offset uint32) error {
	for _, rec := range d.records {
		err := rec.WriteAt(file, offset)
		if err != nil {
			return err
		}
		offset += uint32(rec.RecordLength)
	}
	return nil
}

func (d Directory) GetRecords() []string {
	names := make([]string, 0, len(d.records))
	for name := range d.records {
		names = append(names, name)
	}
	return names
}

func (d Directory) GetInode(recordName string) (uint32, error) {
	record, exist := d.records[recordName]
	if exist {
		return record.Inode, nil
	}
	return 0, fmt.Errorf("record not found - %s", recordName)
}
