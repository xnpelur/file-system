package inode

import (
	"encoding/binary"
	"file-system/internal/filesystem/user"
	"file-system/internal/utils"
	"io"
	"os"
	"strconv"
	"strings"
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

type TypeAndPermissions struct {
	IsFile             bool
	IsHidden           bool
	OwnerReadAccess    bool
	OwnerWriteAccess   bool
	OwnerExecuteAccess bool
	UsersReadAccess    bool
	UsersWriteAccess   bool
	UsersExecuteAccess bool
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

	// if isHidden {
	// 	typeAndPermissionsValue |= 0b01000000
	// }

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

func NewTypeAndPermissions(isFile bool, numericPermissions int) TypeAndPermissions {
	users := numericPermissions % 10
	owner := numericPermissions / 10 % 10

	return TypeAndPermissions{
		IsFile:             isFile,
		IsHidden:           false,
		OwnerReadAccess:    (owner>>2)&1 == 1,
		OwnerWriteAccess:   (owner>>1)&1 == 1,
		OwnerExecuteAccess: (owner>>0)&1 == 1,
		UsersReadAccess:    (users>>2)&1 == 1,
		UsersWriteAccess:   (users>>1)&1 == 1,
		UsersExecuteAccess: (users>>0)&1 == 1,
	}
}

func GetInodeSize() uint32 {
	inodeDummy := Inode{}
	size, _ := utils.CalculateStructSize(inodeDummy)
	return size
}

func UnpackTypeAndPermissions(value uint8) TypeAndPermissions {
	return TypeAndPermissions{
		IsFile:             value&0b10000000 != 0,
		IsHidden:           value&0b01000000 != 0,
		OwnerReadAccess:    value&0b00100000 != 0,
		OwnerWriteAccess:   value&0b00010000 != 0,
		OwnerExecuteAccess: value&0b00001000 != 0,
		UsersReadAccess:    value&0b00000100 != 0,
		UsersWriteAccess:   value&0b00000010 != 0,
		UsersExecuteAccess: value&0b00000001 != 0,
	}
}

func PackTypeAndPermissions(typeAndPermissions TypeAndPermissions) uint8 {
	var value uint8
	if typeAndPermissions.IsFile {
		value |= 0b10000000
	}
	if typeAndPermissions.IsHidden {
		value |= 0b01000000
	}
	if typeAndPermissions.OwnerReadAccess {
		value |= 0b00100000
	}
	if typeAndPermissions.OwnerWriteAccess {
		value |= 0b00010000
	}
	if typeAndPermissions.OwnerExecuteAccess {
		value |= 0b00001000
	}
	if typeAndPermissions.UsersReadAccess {
		value |= 0b00000100
	}
	if typeAndPermissions.UsersWriteAccess {
		value |= 0b00000010
	}
	if typeAndPermissions.UsersExecuteAccess {
		value |= 0b00000001
	}

	return value
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
	t := UnpackTypeAndPermissions(inode.TypeAndPermissions)

	result := []string{"-", "-", "-", "-", "-", "-", "-"}
	if !t.IsFile {
		result[0] = "d"
	}
	if t.OwnerReadAccess {
		result[1] = "r"
	}
	if t.OwnerWriteAccess {
		result[2] = "w"
	}
	if t.OwnerExecuteAccess {
		result[3] = "x"
	}
	if t.UsersReadAccess {
		result[4] = "r"
	}
	if t.UsersWriteAccess {
		result[5] = "w"
	}
	if t.UsersExecuteAccess {
		result[6] = "x"
	}

	return strings.Join(result, "")
}

func (inode *Inode) ChangePermissions(value int) error {
	isFile := UnpackTypeAndPermissions(inode.TypeAndPermissions).IsFile
	tap, err := getTapValue(isFile, value)
	if err != nil {
		return err
	}
	inode.TypeAndPermissions = tap
	return nil
}

func (inode Inode) HasReadPermission(user user.User) bool {
	tap := UnpackTypeAndPermissions(inode.TypeAndPermissions)
	return tap.UsersReadAccess || user.UserId == inode.UserId && tap.OwnerReadAccess
}

func (inode Inode) HasWritePermission(user user.User) bool {
	tap := UnpackTypeAndPermissions(inode.TypeAndPermissions)
	return tap.UsersWriteAccess || user.UserId == inode.UserId && tap.OwnerWriteAccess
}

func (inode Inode) IsFile() bool {
	return UnpackTypeAndPermissions(inode.TypeAndPermissions).IsFile
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
