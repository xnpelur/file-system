package directory

import (
	"encoding/binary"
	"file-system/internal/errs"
	"file-system/internal/filesystem/directory/record"
	"fmt"
)

type Directory struct {
	records map[string]record.Record
	keys    []string
}

func NewDirectory(inode uint32, parentInode uint32) *Directory {
	currDir := record.NewRecord(inode, ".")
	parentDir := record.NewRecord(parentInode, "..")

	records := make(map[string]record.Record)
	records[currDir.Name] = currDir
	records[parentDir.Name] = parentDir

	return &Directory{
		records: records,
		keys:    []string{".", ".."},
	}
}

func ReadDirectoryFromBytes(data []byte) (*Directory, error) {
	directory := Directory{
		records: make(map[string]record.Record),
	}
	offset := 0
	for {
		if len(data) < offset+4 {
			break
		}
		inodeData := data[offset : offset+4]
		inode := binary.BigEndian.Uint32(inodeData)
		offset += 4

		recordLengthData := data[offset : offset+2]
		recordLength := binary.BigEndian.Uint16(recordLengthData)
		if recordLength == 0 {
			break // Empty record was read
		}
		offset += 2

		nameLength := data[offset]
		offset += 1

		nameData := data[offset : offset+int(nameLength)]
		name := string(nameData)
		offset += int(nameLength)

		record := record.Record{
			Inode:        inode,
			RecordLength: recordLength,
			NameLength:   nameLength,
			Name:         name,
		}
		directory.records[record.Name] = record
		directory.keys = append(directory.keys, record.Name)
	}

	return &directory, nil
}

func (d *Directory) AddFile(inode uint32, name string) {
	record := record.NewRecord(inode, name)
	d.records[record.Name] = record
	d.keys = append(d.keys, record.Name)
}

func (d *Directory) DeleteFile(name string) {
	delete(d.records, name)

	var newKeys []string
	for _, v := range d.keys {
		if v != name {
			newKeys = append(newKeys, v)
		}
	}
	d.keys = newKeys
}

func (d Directory) Encode() []byte {
	data := make([]byte, 0)
	for _, key := range d.keys {
		bytes := d.records[key].Encode()
		data = append(data, bytes...)
	}
	return data
}

func (d Directory) GetRecords() []string {
	return d.keys
}

func (d Directory) GetInode(recordName string) (uint32, error) {
	record, exist := d.records[recordName]
	if exist {
		return record.Inode, nil
	}
	return 0, fmt.Errorf("%w - %s", errs.ErrRecordNotFound, recordName)
}
