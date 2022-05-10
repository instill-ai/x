package checkfield_test

import (
	"testing"

	"github.com/instill-ai/x/checkfield"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

func TestCheckRequiredFieldsCreate_NoError(t *testing.T) {
	type A struct {
		Field1 string
	}
	requiredFields := []string{"Field1"}

	msg := &A{
		Field1: "field1",
	}

	err := checkfield.CheckRequiredFieldsCreate(msg, requiredFields)
	require.NoError(t, err)
}

func TestCheckRequiredFieldsCreate_RequiredString(t *testing.T) {
	type A struct {
		Field1 string
	}
	requiredFields := []string{"Field1"}

	msg := new(A)

	err := checkfield.CheckRequiredFieldsCreate(msg, requiredFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = required field `Field1` is not provided")
}

func TestCheckRequiredFieldsCreate_RequiredPtr(t *testing.T) {
	type A struct {
		Field1 *string
	}
	requiredFields := []string{"Field1"}

	msg := new(A)

	err := checkfield.CheckRequiredFieldsCreate(msg, requiredFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = required field `Field1` is not provided")
}

func TestCheckRequiredFieldsCreate_RequiredStructPtr(t *testing.T) {
	type A struct {
		Field1 string
	}
	type B struct {
		Field2 *A
	}
	requiredFields := []string{"Field2"}

	msg := new(B)

	err := checkfield.CheckRequiredFieldsCreate(msg, requiredFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = required field `Field2` is not provided")
}

func TestCheckRequiredFieldsCreate_RequiredNestedValid(t *testing.T) {
	type A struct {
		Field1 string
		Field2 string
	}
	type B struct {
		Field1 *A
	}
	requiredFields := []string{"Field1"}

	msg := &B{
		Field1: &A{
			Field1: "field1_A",
		},
	}

	err := checkfield.CheckRequiredFieldsCreate(msg, requiredFields)
	require.NoError(t, err)
}

func TestCheckRequiredFieldsCreate_RequiredNestedInValid(t *testing.T) {
	type A struct {
		Field1 string
		Field2 string
	}
	type B struct {
		Field1 *A
	}
	requiredFields := []string{"Field1"}

	msg := &B{
		Field1: &A{
			Field2: "field2_A",
		},
	}

	err := checkfield.CheckRequiredFieldsCreate(msg, requiredFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = required field `Field1` is not provided")
}

func TestCheckOutputOnlyFieldsCreate_Valid(t *testing.T) {
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
	}
	outputFields := []string{"FieldBool", "FieldInt", "FieldInt8", "FieldInt16", "FieldInt32", "FieldInt64", "FieldUint", "FieldUint8", "FieldUint16", "FieldUint32", "FieldUint64", "FieldFloat32", "FieldFloat64", "FieldStr", "FieldStrPtr"}

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
	}
	err := checkfield.CheckOutputOnlyFieldsCreate(msg, outputFields)
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
	}, msg)
}

func TestCheckImmutableFieldsUpdate_NoUpdate(t *testing.T) {
	type A struct {
		Field1 string
	}
	immutableFields := []string{"Field1"}

	msgReq := &A{
		Field1: "msgUpdate",
	}
	msgUpdate := &A{
		Field1: "msgUpdate",
	}

	err := checkfield.CheckImmutableFieldsUpdate(msgReq, msgUpdate, immutableFields)
	require.NoError(t, err)
}

func TestCheckImmutableFieldsUpdate_UpdateImmutableBool(t *testing.T) {
	type A struct {
		Field1 bool
	}
	immutableFields := []string{"Field1"}

	msgReq := &A{
		Field1: true,
	}
	msgUpdate := &A{
		Field1: false,
	}

	err := checkfield.CheckImmutableFieldsUpdate(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = field `Field1` is immutable")
}

func TestCheckImmutableFieldsUpdate_UpdateImmutableStr(t *testing.T) {
	type A struct {
		Field1 string
	}
	immutableFields := []string{"Field1"}

	msgReq := &A{
		Field1: "msgReq",
	}
	msgUpdate := &A{
		Field1: "msgUpdate",
	}

	err := checkfield.CheckImmutableFieldsUpdate(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = field `Field1` is immutable")
}

func TestCheckImmutableFieldsUpdate_UpdateImmutableInt(t *testing.T) {
	type A struct {
		Field1 int
	}
	immutableFields := []string{"Field1"}

	msgReq := &A{
		Field1: 10,
	}
	msgUpdate := &A{
		Field1: 20,
	}

	err := checkfield.CheckImmutableFieldsUpdate(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = field `Field1` is immutable")
}

func TestCheckImmutableFieldsUpdate_UpdateImmutableFloat(t *testing.T) {
	type A struct {
		Field1 float32
	}
	immutableFields := []string{"Field1"}

	msgReq := &A{
		Field1: 10,
	}
	msgUpdate := &A{
		Field1: 20,
	}

	err := checkfield.CheckImmutableFieldsUpdate(msgReq, msgUpdate, immutableFields)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = field `Field1` is immutable")
}

func TestCheckOutputOnlyFieldsUpdate_Valid(t *testing.T) {
	mask := new(fieldmaskpb.FieldMask)
	mask.Paths = []string{"snake_case", "a.b", "a.c", "a.b.c", "b.a"}
	outputOnlyFields := []string{"SnakeCase", "A", "C"}
	maskUpdated, err := checkfield.CheckOutputOnlyFieldsUpdate(mask, outputOnlyFields)
	require.NoError(t, err)

	maskExpected := new(fieldmaskpb.FieldMask)
	maskExpected.Paths = []string{"b.a"}

	require.Equal(t, maskExpected, maskUpdated)
}

func TestCheckResourceID_Valid(t *testing.T) {
	err := checkfield.CheckResourceID("local-user")
	require.NoError(t, err)
}

func TestCheckResourceID_InvalidShort(t *testing.T) {
	// 0-charactor string
	tooShort := ""
	err := checkfield.CheckResourceID(tooShort)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = `id` needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
}

func TestCheckResourceID_InvalidLong(t *testing.T) {

	// 64-charactor string
	tooLong := "abcdefghijklmnopqrstuvwxyz-ABCDEFGHIJKLMNOPQRSTUVWXYZ-0123456789"
	err := checkfield.CheckResourceID(tooLong)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = `id` needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
}

func TestCheckResourceID_InvalidUUID(t *testing.T) {
	a := "91be8b99-cd60-4081-9187-9796d01fd50b"
	err := checkfield.CheckResourceID(a)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = `id` is not allowed to be a UUID")
}

func TestCheckResourceID_Invalid(t *testing.T) {
	a := "local_user"
	err := checkfield.CheckResourceID(a)
	require.EqualError(t, err, "rpc error: code = InvalidArgument desc = `id` needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
}
