package checkfield

import (
	"reflect"
	"regexp"

	"github.com/gogo/status"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
)

// CheckRequiredFields implements follows https://google.aip.dev/203#required
func CheckRequiredFields(msg interface{}, requiredFields []string) error {
	for i := 0; i < reflect.Indirect(reflect.ValueOf(msg)).NumField(); i++ {
		fieldName := reflect.Indirect(reflect.ValueOf(msg)).Type().Field(i).Name
		if contains(requiredFields, fieldName) {
			f := reflect.Indirect(reflect.ValueOf(msg)).FieldByName(fieldName)
			switch f.Kind() {
			case reflect.String:
				if f.String() == "" {
					return status.Errorf(codes.InvalidArgument, "Required field %s is not provided", fieldName)
				}
			case reflect.Ptr:
				if f.IsNil() {
					return status.Errorf(codes.InvalidArgument, "Required field %s is not provided", fieldName)
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
func CheckOutputOnlyFields(msg interface{}, outputOnlyFields []string) error {
	for _, field := range outputOnlyFields {
		f := reflect.Indirect(reflect.ValueOf(msg)).FieldByName(field)
		switch f.Kind() {
		case reflect.Int32:
			reflect.ValueOf(msg).Elem().FieldByName(field).SetInt(0)
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
func CheckImmutableFields(msgReq *interface{}, msgUpdate *interface{}, immutableFields []string) error {
	for _, field := range immutableFields {
		f := reflect.Indirect(reflect.ValueOf(msgReq)).FieldByName(field)
		switch f.Kind() {
		case reflect.String:
			if f.String() != "" {
				if f.String() != reflect.Indirect(reflect.ValueOf(msgUpdate)).FieldByName(field).String() {
					return status.Errorf(codes.InvalidArgument, "Field %s is immutable", field)
				}
			}
		}
	}
	return nil
}

// CheckResourceID implements follows https://google.aip.dev/122#resource-id-segments
func CheckResourceID(id string) error {
	if match, _ := regexp.MatchString("^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$", id); !match {
		return status.Error(codes.InvalidArgument, "The id of pipeline needs to be within ASCII-only 63 characters following RFC-1034 with a regexp (^[a-z]([a-z0-9-]{0,61}[a-z0-9])?$)")
	}
	if _, err := uuid.Parse(id); err == nil {
		return status.Error(codes.InvalidArgument, "The id is a UUID, which is not allowed")
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
