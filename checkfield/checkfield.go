package checkfield

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/gogo/status"
	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// CheckRequiredFieldsCreate implements follows https://google.aip.dev/203#required
// The msg parameter is a Protobuf message instance
// The requiredFields is a slice of field path with snake_case name
func CheckRequiredFieldsCreate(msg interface{}, requiredFields []string) error {

	var recurMsgCheck func(interface{}, []string, string) error
	recurMsgCheck = func(m interface{}, fieldNames []string, path string) error {
		f := reflect.Indirect(reflect.ValueOf(m)).FieldByName(strcase.ToCamel(fieldNames[0]))
		switch f.Kind() {
		case reflect.Invalid:
			return status.Errorf(codes.InvalidArgument, "required field path `%s` is not found in the Protobuf message", path)
		case reflect.String:
			if f.String() == "" {
				return status.Errorf(codes.InvalidArgument, "required field path `%s` is not assigned", path)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() == 0 {
				return status.Errorf(codes.InvalidArgument, "required field path `%s` is not assigned", path)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f.Uint() == 0 {
				return status.Errorf(codes.InvalidArgument, "required field path `%s` is not assigned", path)
			}
		case reflect.Float32, reflect.Float64:
			if f.Float() == 0 {
				return status.Errorf(codes.InvalidArgument, "required field path `%s` is not assigned", path)
			}
		case reflect.Struct:
			if len(fieldNames) > 1 {
				path, fieldNames = path+"."+fieldNames[0], fieldNames[1:]
				if err := recurMsgCheck(f.Interface(), fieldNames, path); err != nil {
					return err
				}
			}
		case reflect.Ptr:
			if f.IsNil() {
				return status.Errorf(codes.InvalidArgument, "required field path `%s` is not assigned", path)
			} else if reflect.ValueOf(f).Kind() == reflect.Struct {
				if len(fieldNames) > 1 {
					path, fieldNames = path+"."+fieldNames[0], fieldNames[1:]
					if err := recurMsgCheck(f.Interface(), fieldNames, path); err != nil {
						return err
					}
				}
			}
		}
		return nil
	}

	for _, path := range requiredFields {
		fieldNames := strings.Split(path, ".")
		if err := recurMsgCheck(msg, fieldNames, fieldNames[0]); err != nil {
			return err
		}
	}

	return nil
}

// CheckOutputOnlyFieldsCreate implements follows https://google.aip.dev/203#output-only
// The msg parameter is a Protobuf message instance
// The outputOnlyFields is a slice of field path with snake_case name
func CheckOutputOnlyFieldsCreate(msg interface{}, outputOnlyFields []string) error {

	var recurMsgCheck func(interface{}, []string, string) error
	recurMsgCheck = func(m interface{}, fieldNames []string, path string) error {
		fieldName := strcase.ToCamel(fieldNames[0])
		f := reflect.ValueOf(m).Elem().FieldByName(fieldName)
		switch f.Kind() {
		case reflect.Invalid:
			return status.Errorf(codes.InvalidArgument, "output-only field path `%s` is not found in the Protobuf message", path)
		case reflect.Bool:
			f.SetBool(false)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			f.SetInt(0)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			f.SetUint(uint64(0))
		case reflect.Float32, reflect.Float64:
			f.SetFloat(0)
		case reflect.String:
			f.SetString("")
		case reflect.Ptr:
			if len(fieldNames) > 1 && reflect.ValueOf(f).Kind() == reflect.Struct {
				path, fieldNames = path+"."+fieldNames[0], fieldNames[1:]
				if err := recurMsgCheck(f.Interface(), fieldNames, path); err != nil {
					return err
				}
			} else {
				f.Set(reflect.Zero(f.Type()))
			}
		}
		return nil
	}

	for _, path := range outputOnlyFields {
		fieldNames := strings.Split(path, ".")
		if err := recurMsgCheck(msg, fieldNames, fieldNames[0]); err != nil {
			return err
		}
	}

	return nil
}

// CheckImmutableFieldsUpdate implements follows https://google.aip.dev/203#immutable
// The msgReq parameter is a Protobuf message instance requested to update msgUpdate
// The outputOnlyFields is a slice of field path with snake_case name
func CheckImmutableFieldsUpdate(msgReq interface{}, msgUpdate interface{}, immutableFields []string) error {

	var recurMsgCheck func(interface{}, interface{}, []string, string) error
	recurMsgCheck = func(mr interface{}, mu interface{}, fieldNames []string, path string) error {
		fieldName := strcase.ToCamel(fieldNames[0])
		f := reflect.Indirect(reflect.ValueOf(mr)).FieldByName(fieldName)
		switch f.Kind() {
		case reflect.Invalid:
			return status.Errorf(codes.InvalidArgument, "immutable field path `%s` is not found in the Protobuf message", path)
		case reflect.Bool:
			if f.Bool() {
				if f.Bool() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Bool() {
					return status.Errorf(codes.InvalidArgument, "field path `%s` is immutable", path)
				}
			}
		case reflect.String:
			if f.String() != "" {
				if f.String() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).String() {
					return status.Errorf(codes.InvalidArgument, "field path `%s` is immutable", path)
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() != 0 {
				if f.Int() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Int() {
					return status.Errorf(codes.InvalidArgument, "field path `%v` is immutable", path)
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f.Uint() != 0 {
				if f.Uint() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Uint() {
					return status.Errorf(codes.InvalidArgument, "field path `%v` is immutable", path)
				}
			}
		case reflect.Float32, reflect.Float64:
			if f.Float() != 0 {
				if f.Float() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Float() {
					return status.Errorf(codes.InvalidArgument, "field path `%v` is immutable", path)
				}
			}
		case reflect.Ptr:
			if len(fieldNames) > 1 && reflect.ValueOf(f).Kind() == reflect.Struct {
				path, fieldNames = path+"."+fieldNames[0], fieldNames[1:]
				if err := recurMsgCheck(f.Interface(), reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Interface(), fieldNames, path); err != nil {
					return err
				}
			} else {
				f.Set(reflect.Zero(f.Type()))
			}
		}
		return nil
	}

	for _, path := range immutableFields {
		fieldNames := strings.Split(path, ".")
		if err := recurMsgCheck(msgReq, msgUpdate, fieldNames, fieldNames[0]); err != nil {
			return err
		}
	}

	return nil
}

// CheckOutputOnlyFieldsUpdate removes output only fields from the input field mask
//
// output only fields are in `CamelCase` format and fields in the field mask are in `snake_case` format.
// if a path in the input field mask is nested, such as `a.b`, and the output only fields includes `A`,
// the path will be removed. The unfiltered paths will be stored in the the output field mask
// in the original format.
func CheckOutputOnlyFieldsUpdate(mask *fieldmaskpb.FieldMask, outputOnlyFields []string) (*fieldmaskpb.FieldMask, error) {
	maskUpdated := new(fieldmaskpb.FieldMask)

	for _, path := range mask.GetPaths() {
		if !contains(outputOnlyFields, path) {
			maskUpdated.Paths = append(maskUpdated.Paths, path)
		}
	}
	return maskUpdated, nil
}

// CheckResourceID implements follows https://google.aip.dev/122#resource-id-segments
func CheckResourceID(id string) error {
	if _, err := uuid.Parse(id); err == nil {
		return status.Error(codes.InvalidArgument, "`id` is not allowed to be a UUID")
	}

	if match, _ := regexp.MatchString("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$", id); !match {
		return status.Error(codes.InvalidArgument, "`id` needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
	}
	return nil
}

// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
