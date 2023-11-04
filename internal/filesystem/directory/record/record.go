package record

type Record struct {
	Inode        uint32
	RecordLength uint16
	NameLength   uint8
	Name         string
}

func NewRecord(inode uint32, name string) Record {
	recordInstance := Record{}

	recordInstance.Inode = inode
	recordInstance.RecordLength = uint16(len(name)) + 4 + 2 + 1
	recordInstance.NameLength = uint8(len(name))
	recordInstance.Name = name

	return recordInstance
}
