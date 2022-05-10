package checkfield

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/gogo/status"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// CheckRequiredFields implements follows https://google.aip.dev/203#required
// TODO limitation: can't handle number and struct field
func CheckRequiredFields(msg interface{}, requiredFields []string) error {
	for i := 0; i < reflect.Indirect(reflect.ValueOf(msg)).NumField(); i++ {
		fieldName := reflect.Indirect(reflect.ValueOf(msg)).Type().Field(i).Name
		if contains(requiredFields, fieldName) {
			f := reflect.Indirect(reflect.ValueOf(msg)).FieldByName(fieldName)
			fmt.Println("-----------field ", fieldName, f.Kind())
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					return status.Errorf(codes.InvalidArgument, "required field `%s` is not provided", fieldName)
				}
			case reflect.Ptr:
				fmt.Println("==============Ptr", f)
				if f.IsNil() {
					return status.Errorf(codes.InvalidArgument, "required field `%s` is not provided", fieldName)
				} else if reflect.Indirect(reflect.ValueOf(f)).Kind() == reflect.Struct {
					if err := CheckRequiredFields(f.Interface(), requiredFields); err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

// CheckOutputOnlyFields implements follows https://google.aip.dev/203#output-only
// TODO Limitation: can't handle struct
func CheckOutputOnlyFields(msg interface{}, outputOnlyFields []string) error {
	for _, field := range outputOnlyFields {
		f := reflect.Indirect(reflect.ValueOf(msg)).FieldByName(field)
		switch f.Kind() {
		case reflect.Bool:
			reflect.ValueOf(msg).Elem().FieldByName(field).SetBool(false)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			reflect.ValueOf(msg).Elem().FieldByName(field).SetInt(0)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			reflect.ValueOf(msg).Elem().FieldByName(field).SetUint(uint64(0))
		case reflect.Float32, reflect.Float64:
			reflect.ValueOf(msg).Elem().FieldByName(field).SetFloat(0)
		case reflect.String:
			reflect.ValueOf(msg).Elem().FieldByName(field).SetString("")
		case reflect.Ptr:
			reflect.ValueOf(msg).Elem().FieldByName(field).Set(reflect.Zero(f.Type()))
		case reflect.Struct:
			if err := CheckOutputOnlyFields(f, outputOnlyFields); err != nil {
				return err
			}
		}
	}
	return nil
}

// CheckImmutableFields implements follows https://google.aip.dev/203#immutable
// TODO Limitation: can't handle struct or pointer field
func CheckImmutableFields(msgReq interface{}, msgUpdate interface{}, immutableFields []string) error {
	for _, field := range immutableFields {
		f := reflect.Indirect(reflect.ValueOf(msgReq)).FieldByName(field)
		switch f.Kind() {
		case reflect.Bool:
			if f.Bool() {
				if f.Bool() != reflect.Indirect(reflect.ValueOf(msgUpdate)).FieldByName(field).Bool() {
					return status.Errorf(codes.InvalidArgument, "field `%s` is immutable", field)
				}
			}
		case reflect.String:
			if f.String() != "" {
				if f.String() != reflect.Indirect(reflect.ValueOf(msgUpdate)).FieldByName(field).String() {
					return status.Errorf(codes.InvalidArgument, "field `%s` is immutable", field)
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() != 0 {
				if f.Int() != reflect.Indirect(reflect.ValueOf(msgUpdate)).FieldByName(field).Int() {
					return status.Errorf(codes.InvalidArgument, "field `%v` is immutable", field)
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f.Uint() != 0 {
				if f.Uint() != reflect.Indirect(reflect.ValueOf(msgUpdate)).FieldByName(field).Uint() {
					return status.Errorf(codes.InvalidArgument, "field `%v` is immutable", field)
				}
			}
		case reflect.Float32, reflect.Float64:
			if f.Float() != 0 {
				if f.Float() != reflect.Indirect(reflect.ValueOf(msgUpdate)).FieldByName(field).Float() {
					return status.Errorf(codes.InvalidArgument, "field `%v` is immutable", field)
				}
			}
		}
	}
	return nil
}

// CheckOutputOnlyFieldsUpdate removes output only fields from the input field mask
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
