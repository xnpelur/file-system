package inode

import (
	"file-system/internal/utils"
)

type Inode struct {
	TypeAndPermissions uint16
	UserId             uint16
	GroupId            uint16
	FileSize           uint32
	CreationTime       uint32
	ModificationTime   uint32
	LinkCount          uint16
	FileData           [12]uint32
}

type TypeAndPermissions struct {
	IsFile             bool
	OwnerReadAccess    bool
	OwnerWriteAccess   bool
	OwnerExecuteAccess bool
	GroupReadAccess    bool
	GroupWriteAccess   bool
	GroupExecuteAccess bool
	UsersReadAccess    bool
	UsersWriteAccess   bool
	UsersExecuteAccess bool
}

func GetInodeSize() int {
	inodeDummy := Inode{}
	size, _ := utils.CalculateStructSize(inodeDummy)
	return size
}

func UnpackTypeAndPermissions(value uint16) TypeAndPermissions {
	return TypeAndPermissions{
		IsFile:             value&0b1000000000000000 != 0,
		OwnerReadAccess:    value&0b0100000000000000 != 0,
		OwnerWriteAccess:   value&0b0010000000000000 != 0,
		OwnerExecuteAccess: value&0b0001000000000000 != 0,
		GroupReadAccess:    value&0b0000100000000000 != 0,
		GroupWriteAccess:   value&0b0000010000000000 != 0,
		GroupExecuteAccess: value&0b0000001000000000 != 0,
		UsersReadAccess:    value&0b0000000100000000 != 0,
		UsersWriteAccess:   value&0b0000000010000000 != 0,
		UsersExecuteAccess: value&0b0000000001000000 != 0,
	}
}

func PackTypeAndPermissions(typeAndPermissions TypeAndPermissions) uint16 {
	var value uint16
	if typeAndPermissions.IsFile {
		value |= 0b1000000000000000
	}
	if typeAndPermissions.OwnerReadAccess {
		value |= 0b0100000000000000
	}
	if typeAndPermissions.OwnerWriteAccess {
		value |= 0b0010000000000000
	}
	if typeAndPermissions.OwnerExecuteAccess {
		value |= 0b0001000000000000
	}
	if typeAndPermissions.GroupReadAccess {
		value |= 0b0000100000000000
	}
	if typeAndPermissions.GroupWriteAccess {
		value |= 0b0000010000000000
	}
	if typeAndPermissions.GroupExecuteAccess {
		value |= 0b0000001000000000
	}
	if typeAndPermissions.UsersReadAccess {
		value |= 0b0000000100000000
	}
	if typeAndPermissions.UsersWriteAccess {
		value |= 0b0000000010000000
	}
	if typeAndPermissions.UsersExecuteAccess {
		value |= 0b0000000001000000
	}

	return value
}
