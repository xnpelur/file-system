package inode

import (
	"fmt"
	"testing"
)

func TestUnpackTypeAndPermissions(t *testing.T) {
	tests := []struct {
		input uint8
		want  TypeAndPermissions
	}{
		{0b10000000, TypeAndPermissions{IsFile: true}},
		{0b00111000, TypeAndPermissions{OwnerReadAccess: true, OwnerWriteAccess: true, OwnerExecuteAccess: true}},
		{0b00000001, TypeAndPermissions{UsersExecuteAccess: true}},
		{0b00000000, TypeAndPermissions{}},
	}

	for _, test := range tests {
		got := UnpackTypeAndPermissions(test.input)
		if got != test.want {
			t.Errorf("UnpackTypeAndPermissions(%08b) = %v; want %v", test.input, got, test.want)
		}
	}
}

func TestPackTypeAndPermissions(t *testing.T) {
	tests := []struct {
		input TypeAndPermissions
		want  uint8
	}{
		{TypeAndPermissions{IsFile: true}, 0b10000000},
		{TypeAndPermissions{OwnerReadAccess: true, OwnerWriteAccess: true, OwnerExecuteAccess: true}, 0b00111000},
		{TypeAndPermissions{UsersExecuteAccess: true}, 0b00000001},
		{TypeAndPermissions{}, 0b00000000},
	}

	for _, test := range tests {
		got := PackTypeAndPermissions(test.input)
		if got != test.want {
			t.Errorf("PackTypeAndPermissions(%v) = %08b; want %016b", test.input, got, test.want)
		}
	}
}

func TestConversionRoundTrip(t *testing.T) {
	tests := []TypeAndPermissions{
		{IsFile: true, OwnerReadAccess: true},
		{OwnerWriteAccess: true, UsersExecuteAccess: true},
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

func TestNewTypeAndPermissions(t *testing.T) {
	testCases := []struct {
		isFile              bool
		numericPermissions  int
		expectedPermissions TypeAndPermissions
	}{
		{
			isFile:             true,
			numericPermissions: 75,
			expectedPermissions: TypeAndPermissions{
				IsFile:             true,
				OwnerReadAccess:    true,
				OwnerWriteAccess:   true,
				OwnerExecuteAccess: true,
				UsersReadAccess:    true,
				UsersWriteAccess:   false,
				UsersExecuteAccess: true,
			},
		},
		{
			isFile:             false,
			numericPermissions: 64,
			expectedPermissions: TypeAndPermissions{
				IsFile:             false,
				OwnerReadAccess:    true,
				OwnerWriteAccess:   true,
				OwnerExecuteAccess: false,
				UsersReadAccess:    true,
				UsersWriteAccess:   false,
				UsersExecuteAccess: false,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("isFile=%v, numericPermissions=%d", tc.isFile, tc.numericPermissions), func(t *testing.T) {
			actualPermissions := NewTypeAndPermissions(tc.isFile, tc.numericPermissions)

			if actualPermissions != tc.expectedPermissions {
				t.Errorf("Expected %+v, but got %+v", tc.expectedPermissions, actualPermissions)
			}
		})
	}
}

func Test(t *testing.T) {
	testCases := []struct {
		i        Inode
		expected string
	}{
		{
			i: Inode{
				TypeAndPermissions: PackTypeAndPermissions(
					TypeAndPermissions{
						IsFile:             false,
						OwnerReadAccess:    true,
						OwnerWriteAccess:   true,
						OwnerExecuteAccess: true,
						UsersReadAccess:    true,
						UsersWriteAccess:   true,
						UsersExecuteAccess: true,
					},
				),
			},
			expected: "drwxrwx",
		},
		{
			i: Inode{
				TypeAndPermissions: PackTypeAndPermissions(
					TypeAndPermissions{
						IsFile:             true,
						OwnerReadAccess:    true,
						OwnerWriteAccess:   false,
						OwnerExecuteAccess: true,
						UsersReadAccess:    true,
						UsersWriteAccess:   false,
						UsersExecuteAccess: true,
					},
				),
			},
			expected: "-r-xr-x",
		},
		{
			i: Inode{
				TypeAndPermissions: PackTypeAndPermissions(
					TypeAndPermissions{
						IsFile:             true,
						OwnerReadAccess:    true,
						OwnerWriteAccess:   true,
						OwnerExecuteAccess: true,
						UsersReadAccess:    false,
						UsersWriteAccess:   false,
						UsersExecuteAccess: false,
					},
				),
			},
			expected: "-rwx---",
		},
	}

	for _, testCase := range testCases {
		result := testCase.i.GetTypeAndPermissionString()
		if result != testCase.expected {
			t.Errorf("Expected: %s, Got: %s", testCase.expected, result)
		}
	}
}
