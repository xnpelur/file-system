package inode

import (
	"testing"
)

func TestUnpackTypeAndPermissions(t *testing.T) {
	tests := []struct {
		input uint16
		want  TypeAndPermissions
	}{
		{0b1000000000000000, TypeAndPermissions{IsFile: true}},
		{0b0111100000000000, TypeAndPermissions{OwnerReadAccess: true, OwnerWriteAccess: true, OwnerExecuteAccess: true, GroupReadAccess: true}},
		{0b0000000001000000, TypeAndPermissions{UsersExecuteAccess: true}},
		{0b0000000000000000, TypeAndPermissions{}},
	}

	for _, test := range tests {
		got := UnpackTypeAndPermissions(test.input)
		if got != test.want {
			t.Errorf("UnpackTypeAndPermissions(%016b) = %v; want %v", test.input, got, test.want)
		}
	}
}

func TestPackTypeAndPermissions(t *testing.T) {
	tests := []struct {
		input TypeAndPermissions
		want  uint16
	}{
		{TypeAndPermissions{IsFile: true}, 0b1000000000000000},
		{TypeAndPermissions{OwnerReadAccess: true, OwnerWriteAccess: true, OwnerExecuteAccess: true, GroupReadAccess: true}, 0b0111100000000000},
		{TypeAndPermissions{UsersExecuteAccess: true}, 0b0000000001000000},
		{TypeAndPermissions{}, 0b0000000000000000},
	}

	for _, test := range tests {
		got := PackTypeAndPermissions(test.input)
		if got != test.want {
			t.Errorf("PackTypeAndPermissions(%v) = %016b; want %016b", test.input, got, test.want)
		}
	}
}

func TestConversionRoundTrip(t *testing.T) {
	tests := []TypeAndPermissions{
		{IsFile: true, OwnerReadAccess: true},
		{OwnerWriteAccess: true, GroupReadAccess: true, UsersExecuteAccess: true},
		{OwnerReadAccess: true, UsersReadAccess: true, UsersWriteAccess: true},
		{IsFile: false},
	}

	for _, original := range tests {
		converted := UnpackTypeAndPermissions(PackTypeAndPermissions(original))
		if converted != original {
			t.Errorf("Conversion round trip failed for %v, got %v", original, converted)
		}
	}
}
