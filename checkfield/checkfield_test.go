package checkfield_test

import (
	"testing"

	"github.com/instill-ai/x/checkfield"
	"github.com/stretchr/testify/require"
)

func TestCheckRequiredFields_NoError(t *testing.T) {

	type A struct {
		A struct {
			B struct {
				C string
			}
		}
		D string
	}

	msg := new(A)
	msg.A.B.C = "some nested field"
	msg.D = "some nested field"

	requiredFields := []string{"a.b.c", "d", "a.b", "a"}

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.NoError(t, err)

}

func TestCheckRequiredFields_Error(t *testing.T) {

	type A struct {
		A struct {
			B struct {
				C string
			}
		}
		D string
	}

	msg := new(A)

	requiredFields := []string{"a.b.c", "d", "a.b", "a"}

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.Error(t, err)

}

func TestCheckRequiredFields_RequiredString(t *testing.T) {
	type A struct {
		Field1 string
	}
	requiredFields := []string{"field1"}

	msg := new(A)

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.EqualError(t, err, "required field path `field1` is not assigned")
}

func TestCheckRequiredFields_RequiredPtr(t *testing.T) {
	type A struct {
		Field1 *string
	}
	requiredFields := []string{"field1"}

	msg := new(A)

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.EqualError(t, err, "required field path `field1` is not assigned")
}

func TestCheckRequiredFields_RequiredStructPtr(t *testing.T) {
	type A struct {
		Field1 string
	}
	type B struct {
		Field2 *A
	}
	requiredFields := []string{"field2"}

	msg := new(B)

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.EqualError(t, err, "required field path `field2` is not assigned")
}

func TestCheckRequiredFields_RequiredNestedValid(t *testing.T) {
	type A struct {
		Field1 string
		Field2 string
	}
	type B struct {
		Field1 *A
	}
	requiredFields := []string{"field1"}

	msg := &B{
		Field1: &A{
			Field1: "field1_A",
		},
	}

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.NoError(t, err)
}

func TestCheckRequiredFields_RequiredNestedInValid(t *testing.T) {
	type A struct {
		Field1 string
		Field2 string
	}
	type B struct {
		Field1 *A
	}
	requiredFields := []string{"field1.field1"}

	msg := &B{
		Field1: &A{
			Field2: "field2_A",
		},
	}

	err := checkfield.CheckRequiredFields(msg, requiredFields)
	require.EqualError(t, err, "required field path `field1.field1` is not assigned")
}

func TestCheckCreateOutputOnlyFields_Valid(t *testing.T) {

	type InnerStruct struct {
		FieldStr string
	}

	type A struct {
		FieldBool                bool
		FieldInt                 int
		FieldInt8                int8
		FieldInt16               int16
		FieldInt32               int32
		FieldInt64               int64
		FieldUint                uint
		FieldUint8               uint8
		FieldUint16              uint16
		FieldUint32              uint32
		FieldUint64              uint64
		FieldFloat32             float32
		FieldFloat64             float64
		FieldStr                 string
		FieldStrPtr              *string
		FieldStrNotOutputOnly    string
		FieldStrPtrNotOutputOnly *string
		FieldNestedStruct        *InnerStruct
	}
	outputFields := []string{
		"field_bool",
		"field_int", "field_int8", "field_int16", "field_int32", "field_int64",
		"field_uint", "field_uint8", "field_uint16", "field_uint32", "field_uint64",
		"field_float32", "field_float64",
		"field_str", "field_str_ptr",
		"field_nested_struct.field_str",
	}

	nonEmptyStr := "field"
	msg := &A{
		FieldBool:                true,
		FieldInt:                 10,
		FieldInt8:                10,
		FieldInt16:               10,
		FieldInt32:               10,
		FieldInt64:               10,
		FieldUint:                10,
		FieldUint8:               10,
		FieldUint16:              10,
		FieldUint32:              10,
		FieldUint64:              10,
		FieldFloat32:             10,
		FieldFloat64:             10,
		FieldStr:                 "field_str",
		FieldStrPtr:              &nonEmptyStr,
		FieldStrNotOutputOnly:    "field_str_not_output_only",
		FieldStrPtrNotOutputOnly: &nonEmptyStr,
		FieldNestedStruct: &InnerStruct{
			FieldStr: nonEmptyStr,
		},
	}

	err := checkfield.CheckCreateOutputOnlyFields(msg, outputFields)
	require.NoError(t, err)
	require.Equal(t, &A{
		FieldBool:                false,
		FieldInt:                 0,
		FieldInt8:                0,
		FieldInt16:               0,
		FieldInt32:               0,
		FieldInt64:               0,
		FieldUint:                0,
		FieldUint8:               0,
		FieldUint16:              0,
		FieldUint32:              0,
		FieldUint64:              0,
		FieldFloat32:             0,
		FieldFloat64:             0,
		FieldStr:                 "",
		FieldStrPtr:              nil,
		FieldStrPtrNotOutputOnly: &nonEmptyStr,
		FieldStrNotOutputOnly:    "field_str_not_output_only",
		FieldNestedStruct: &InnerStruct{
			FieldStr: "",
		},
	}, msg)
}

func TestCheckUpdateImmutableFields_NoUpdate(t *testing.T) {
	type A struct {
		Field1 string
	}
	immutableFields := []string{"field1"}

	msgReq := &A{
		Field1: "msgUpdate",
	}
	msgUpdate := &A{
		Field1: "msgUpdate",
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.NoError(t, err)
}

func TestCheckUpdateImmutableFields_UpdateImmutableBool(t *testing.T) {
	type A struct {
		Field1 bool
	}
	immutableFields := []string{"field1"}

	msgReq := &A{
		Field1: true,
	}
	msgUpdate := &A{
		Field1: false,
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "field path `field1` is immutable")
}

func TestCheckUpdateImmutableFields_UpdateImmutableStr(t *testing.T) {
	type A struct {
		Field1 string
	}
	immutableFields := []string{"field1"}

	msgReq := &A{
		Field1: "msgReq",
	}
	msgUpdate := &A{
		Field1: "msgUpdate",
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "field path `field1` is immutable")
}

func TestCheckUpdateImmutableFields_UpdateImmutableInt(t *testing.T) {
	type A struct {
		Field1 int
	}
	immutableFields := []string{"field1"}

	msgReq := &A{
		Field1: 10,
	}
	msgUpdate := &A{
		Field1: 20,
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "field path `field1` is immutable")
}

func TestCheckUpdateImmutableFields_UpdateImmutableFloat(t *testing.T) {
	type A struct {
		Field1 float32
	}
	immutableFields := []string{"field1"}

	msgReq := &A{
		Field1: 10,
	}
	msgUpdate := &A{
		Field1: 20,
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "field path `field1` is immutable")
}

func TestCheckUpdateImmutableFields_UpdateImmutableStruct(t *testing.T) {
	type A struct {
		Field1 float32
	}
	type B struct {
		Field1 *A
	}

	immutableFields := []string{"field1"}

	msgReq := &B{
		Field1: &A{
			Field1: 10,
		},
	}
	msgUpdate := &B{
		Field1: &A{
			Field1: 20,
		},
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "field path `field1` is immutable")
}

func TestCheckUpdateImmutableFields_UpdateImmutableNestedStruct(t *testing.T) {
	type A struct {
		Field1 float32
	}
	type B struct {
		Field1 *A
	}

	immutableFields := []string{"field1.field1"}

	msgReq := &B{
		Field1: &A{
			Field1: 10,
		},
	}
	msgUpdate := &B{
		Field1: &A{
			Field1: 20,
		},
	}

	err := checkfield.CheckUpdateImmutableFields(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "field path `field1.field1` is immutable")
}

func TestCheckResourceID_Valid(t *testing.T) {
	err := checkfield.CheckResourceID("local_user")
	require.NoError(t, err)
}

func TestCheckResourceID_InvalidShort(t *testing.T) {
	// 0-charactor string
	tooShort := ""
	err := checkfield.CheckResourceID(tooShort)
	require.EqualError(t, err, "the ID must consist only of lowercase letters, numbers, or underscores, and its length cannot exceed 32 characters")
}

func TestCheckResourceID_InvalidLong(t *testing.T) {

	// 64-charactor string
	tooLong := "abcdefghijklmnopqrstuvwxyz-ABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789"
	err := checkfield.CheckResourceID(tooLong)
	require.EqualError(t, err, "the ID must consist only of lowercase letters, numbers, or underscores, and its length cannot exceed 32 characters")
}

func TestCheckResourceID_InvalidUUID(t *testing.T) {
	a := "91be8b99-cd60-4081-9187-9796d01fd50b"
	err := checkfield.CheckResourceID(a)
	require.EqualError(t, err, "`id` is not allowed to be a UUID")
}

func TestCheckResourceID_Invalid(t *testing.T) {
	a := "local-user"
	err := checkfield.CheckResourceID(a)
	require.EqualError(t, err, "the ID must consist only of lowercase letters, numbers, or underscores, and its length cannot exceed 32 characters")
}
