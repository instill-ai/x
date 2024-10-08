package structutil

import (
	"encoding/json"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func BytesToInterface(bytes []byte) (interface{}, error) {
	var obj map[string]interface{}
	if err := json.Unmarshal(bytes, &obj); err != nil {
		if err.Error() == "json: cannot unmarshal array into Go value of type map[string]interface {}" {
			var objArray []map[string]interface{}
			if err := json.Unmarshal(bytes, &objArray); err != nil {
				return nil, err
			}
			return objArray, nil
		}
	}

	return obj, nil
}

// Deprecated: use structpb.NewStruct() directly
func MapToProtobufStruct(m map[string]interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(m)
}

func ProtobufStructToMap(s *structpb.Struct) (map[string]interface{}, error) {
	b, err := protojson.Marshal(s)
	if err != nil {
		return nil, err
	}
	m := make(map[string]interface{})
	err = json.Unmarshal(b, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func StructToProtobufStruct(s interface{}) (*structpb.Struct, error) {
	return structpb.NewStruct(s.(map[string]interface{}))
}

func ProtobufStructToStruct(s *structpb.Struct) (interface{}, error) {
	return ProtobufStructToMap(s)
}
