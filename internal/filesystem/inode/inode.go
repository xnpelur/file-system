package inode

import (
	"encoding/binary"
	"file-system/internal/filesystem/user"
	"file-system/internal/utils"
	"io"
	"os"
	"strconv"
)

type Inode struct {
	TypeAndPermissions uint8
	UserId             uint16
	GroupId            uint16
	FileSize           uint32
	CreationTime       uint32
	ModificationTime   uint32
	Blocks             [12]uint32
}

func NewInode(
	isFile bool,
	numericPermissions int,
	userId int,
	groupId int,
	dataBlocks []uint32,
) (*Inode, error) {
	var blocks [12]uint32
	for i, dataBlock := range dataBlocks {
		if i >= 12 {
			break
		}
		blocks[i] = dataBlock
	}

	tap, err := getTapValue(isFile, numericPermissions)
	if err != nil {
		return nil, err
	}

	return &Inode{
		TypeAndPermissions: tap,
		UserId:             uint16(userId),
		GroupId:            uint16(groupId),
		FileSize:           uint32(len(dataBlocks)),
		Blocks:             blocks,
	}, nil
}

func getTapValue(isFile bool, numericPermissions int) (uint8, error) {
	strNumber := strconv.FormatInt(int64(numericPermissions), 10)
	decimalNumber, err := strconv.ParseUint(strNumber, 8, 8)
	if err != nil {
		return 0, err
	}

	value := uint8(decimalNumber)
	if isFile {
		value |= 0b10000000
	}

	return value, nil
}

func GetInodeSize() uint32 {
	inodeDummy := Inode{}
	size, _ := utils.CalculateStructSize(inodeDummy)
	return size
}

func ReadInodeAt(file *os.File, offset uint32) (*Inode, error) {
	_, err := file.Seek(int64(offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	data := make([]byte, 65)
	_, err = file.Read(data)
	if err != nil {
		return nil, err
	}

	return decodeInode(data), nil
}

func decodeInode(data []byte) *Inode {
	inode := Inode{}

	inode.TypeAndPermissions = data[0]
	inode.UserId = binary.BigEndian.Uint16(data[1:3])
	inode.GroupId = binary.BigEndian.Uint16(data[3:5])
	inode.FileSize = binary.BigEndian.Uint32(data[5:9])
	inode.CreationTime = binary.BigEndian.Uint32(data[9:13])
	inode.ModificationTime = binary.BigEndian.Uint32(data[13:17])

	for i := 0; i < 12; i++ {
		offset := 17 + i*4
		inode.Blocks[i] = binary.BigEndian.Uint32(data[offset : offset+4])
	}

	return &inode
}

func (inode Inode) GetTypeAndPermissionString() string {
	permissions := "rwx"

	result := []byte("-------")
	if !inode.IsFile() {
		result[0] = 'd'
	}

	for i := 0; i < 6; i++ {
		if int(inode.TypeAndPermissions)>>(5-i)&1 == 1 {
			result[i+1] = permissions[i%3]
		}
	}

	return string(result)
}

func (inode *Inode) ChangePermissions(value int) error {
	tap, err := getTapValue(inode.IsFile(), value)
	if err != nil {
		return err
	}
	inode.TypeAndPermissions = tap
	return nil
}

func (inode Inode) HasReadPermission(user user.User) bool {
	ownerReadAccess := inode.TypeAndPermissions&0b00100000 != 0
	usersReadAccess := inode.TypeAndPermissions&0b00000100 != 0

	return usersReadAccess || user.UserId == inode.UserId && ownerReadAccess
}

func (inode Inode) HasWritePermission(user user.User) bool {
	ownerWriteAccess := inode.TypeAndPermissions&0b00010000 != 0
	usersWriteAccess := inode.TypeAndPermissions&0b00000010 != 0

	return usersWriteAccess || user.UserId == inode.UserId && ownerWriteAccess
}

func (inode Inode) IsFile() bool {
	return inode.TypeAndPermissions&0b10000000 != 0
}

func (inode Inode) WriteAt(file *os.File, offset uint32) error {
	data := inode.encode()

	_, err := file.WriteAt(data, int64(offset))
	if err != nil {
		return err
	}

	return nil
}

func (value Inode) encode() []byte {
	data := make([]byte, 65)

	data[0] = value.TypeAndPermissions
	binary.BigEndian.PutUint16(data[1:3], value.UserId)
	binary.BigEndian.PutUint16(data[3:5], value.GroupId)
	binary.BigEndian.PutUint32(data[5:9], value.FileSize)
	binary.BigEndian.PutUint32(data[9:13], value.CreationTime)
	binary.BigEndian.PutUint32(data[13:17], value.ModificationTime)

	for i := 0; i < 12; i++ {
		offset := 17 + i*4
		binary.BigEndian.PutUint32(data[offset:offset+4], value.Blocks[i])
	}

	return data
}
