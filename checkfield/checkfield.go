package checkfield

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/iancoleman/strcase"
	"google.golang.org/protobuf/types/known/fieldmaskpb"
)

// CheckRequiredFields implements follows https://google.aip.dev/203#required
// The msg parameter is a Protobuf message instance
// The requiredFields is a slice of field path with snake_case name
func CheckRequiredFields(msg any, requiredFields []string) error {

	var recurMsgCheck func(any, []string, string) error
	recurMsgCheck = func(m any, fieldNames []string, path string) error {

		if reflect.ValueOf(m).IsZero() {
			return fmt.Errorf("required field path `%s` is empty", path)
		}

		f := reflect.Indirect(reflect.ValueOf(m)).FieldByName(strcase.ToCamel(fieldNames[0]))
		switch f.Kind() {
		case reflect.Invalid:
			return fmt.Errorf("required field path `%s` is not found in the Protobuf message", path)
		case reflect.String:
			if f.String() == "" {
				return fmt.Errorf("required field path `%s` is not assigned", path)
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if f.Int() == 0 {
				return fmt.Errorf("required field path `%s` is not assigned", path)
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if f.Uint() == 0 {
				return fmt.Errorf("required field path `%s` is not assigned", path)
			}
		case reflect.Float32, reflect.Float64:
			if f.Float() == 0 {
				return fmt.Errorf("required field path `%s` is not assigned", path)
			}
		case reflect.Struct:
			if len(fieldNames) > 1 {
				path, fieldNames = path+"."+fieldNames[1], fieldNames[1:]
				if err := recurMsgCheck(f.Interface(), fieldNames, path); err != nil {
					return err
				}
			}
		case reflect.Ptr:
			if f.IsNil() {
				return fmt.Errorf("required field path `%s` is not assigned", path)
			} else if len(fieldNames) > 1 && reflect.ValueOf(f).Kind() == reflect.Struct {
				path, fieldNames = path+"."+fieldNames[1], fieldNames[1:]
				if err := recurMsgCheck(f.Interface(), fieldNames, path); err != nil {
					return err
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

// CheckCreateOutputOnlyFields implements follows https://google.aip.dev/203#output-only
// The msg parameter is a Protobuf message instance
// The outputOnlyFields is a slice of field path with snake_case name
func CheckCreateOutputOnlyFields(msg any, outputOnlyFields []string) error {

	var recurMsgCheck func(any, []string, string) error
	recurMsgCheck = func(m any, fieldNames []string, path string) error {

		if reflect.ValueOf(m).IsZero() {
			return fmt.Errorf("output-only field path `%s` is empty", path)
		}

		fieldName := strcase.ToCamel(fieldNames[0])
		f := reflect.ValueOf(m).Elem().FieldByName(fieldName)
		switch f.Kind() {
		case reflect.Invalid:
			return fmt.Errorf("output-only field path `%s` is not found in the Protobuf message", path)
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
			if !f.IsNil() && len(fieldNames) > 1 && reflect.ValueOf(f).Kind() == reflect.Struct {
				path, fieldNames = path+"."+fieldNames[1], fieldNames[1:]
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

// CheckUpdateImmutableFields implements follows https://google.aip.dev/203#immutable
// The msgReq parameter is a Protobuf message instance requested to update msgUpdate
// The outputOnlyFields is a slice of field path with snake_case name
func CheckUpdateImmutableFields(msgReq any, msgUpdate any, immutableFields []string) error {

	var recurMsgCheck func(any, any, []string, string) error
	recurMsgCheck = func(mr any, mu any, fieldNames []string, path string) error {

		if reflect.ValueOf(mr).IsZero() {
			return fmt.Errorf("immutable field path `%s` in request message is empty", path)
		} else if reflect.ValueOf(mu).IsZero() {
			return fmt.Errorf("immutable field path `%s` in update message is empty", path)
		}

		fieldName := strcase.ToCamel(fieldNames[0])
		f := reflect.Indirect(reflect.ValueOf(mr)).FieldByName(fieldName)
		switch f.Kind() {
		case reflect.Invalid:
			return fmt.Errorf("immutable field path `%s` is not found in the Protobuf message", path)
		case reflect.Bool:
			if !f.IsZero() {
				if f.Bool() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Bool() {
					return fmt.Errorf("field path `%s` is immutable", path)
				}
			}
		case reflect.String:
			if !f.IsZero() {
				if f.String() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).String() {
					return fmt.Errorf("field path `%s` is immutable", path)
				}
			}
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			if !f.IsZero() {
				if f.Int() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Int() {
					return fmt.Errorf("field path `%v` is immutable", path)
				}
			}
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			if !f.IsZero() {
				if f.Uint() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Uint() {
					return fmt.Errorf("field path `%v` is immutable", path)
				}
			}
		case reflect.Float32, reflect.Float64:
			if !f.IsZero() {
				if f.Float() != reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Float() {
					return fmt.Errorf("field path `%v` is immutable", path)
				}
			}
		case reflect.Ptr:
			if !f.IsZero() {
				if len(fieldNames) > 1 && reflect.ValueOf(f).Kind() == reflect.Struct {
					path, fieldNames = path+"."+fieldNames[1], fieldNames[1:]
					if err := recurMsgCheck(f.Interface(), reflect.Indirect(reflect.ValueOf(mu)).FieldByName(fieldName).Interface(), fieldNames, path); err != nil {
						return err
					}
				} else {
					return fmt.Errorf("field path `%v` is immutable", path)
				}
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

// CheckUpdateOutputOnlyFields removes outputOnlyFields from the input field mask
// outputOnlyFields are field paths in `snake_case` like paths in the input field mask.
func CheckUpdateOutputOnlyFields(mask *fieldmaskpb.FieldMask, outputOnlyFields []string) (*fieldmaskpb.FieldMask, error) {
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
		return fmt.Errorf("`id` is not allowed to be a UUID")
	}

	if match, _ := regexp.MatchString("^[a-z_][-a-z_0-9]{0,31}$", id); !match {
		return fmt.Errorf("the ID must start with a lowercase letter or underscore, followed by zero to 31 occurrences of lowercase letters, numbers, hyphens, or underscores")
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
