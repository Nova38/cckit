package serialize

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type (
	SuppportedType string
	TargetType     string

	GenericSerializer struct {
		Supports        []SuppportedType
		Target          TargetType
		UseProtoNames   bool // to use proto field name instead of lowerCamelCase name in JSON field names
		UseEnumNumbers  bool // to use numbers instead of strings for enums
		EmitUnpopulated bool // Emit the default value for unpopulated feilds

	}

	StringSerializer struct {
	}

	BinarySerializer struct {
	}

	JSONSerializer struct {
	}
)

var (
	// AnyType defines that serializer supports any supportable variable type
	AnyType SuppportedType = `any`
	// StructType defines that serializer supports struct type
	StructType SuppportedType = `struct`
	ProtoType  SuppportedType = `proto`
	ScalarType SuppportedType = `scalar`

	DefaultTarget TargetType = `default`
	PreferJSON    TargetType = `json`

	DefaultSerializer = &GenericSerializer{
		Supports:      []SuppportedType{AnyType},
		Target:        DefaultTarget,
		UseProtoNames: true,
	}

	PreferJSONSerializer = &GenericSerializer{
		Supports:        []SuppportedType{AnyType},
		Target:          PreferJSON,
		UseProtoNames:   true,
		UseEnumNumbers:  true,
		EmitUnpopulated: true,
	}
	KeySerializer = &StringSerializer{}

	ErrOnlyStringSupported = errors.New(`only string supported`)
)

func (g *GenericSerializer) ToBytesFrom(entry interface{}) ([]byte, error) {
	switch entryType := entry.(type) {
	case Serializable:
		return entryType.ToBytes(g)

	case proto.Message:
		if g.Target == PreferJSON {
			mo := &protojson.MarshalOptions{UseProtoNames: g.UseProtoNames, UseEnumNumbers: g.UseEnumNumbers, EmitUnpopulated: g.EmitUnpopulated}
			return JSONProtoMarshal(entryType, mo)
		} else {
			return BinaryProtoMarshal(entryType)
		}
	default:
		return toBytes(entry)
	}

}

func (g *GenericSerializer) FromBytesTo(serialized []byte, target interface{}) (interface{}, error) {
	switch targetType := target.(type) {

	case proto.Message:
		if g.Target == PreferJSON {
			return JSONProtoUnmarshal(serialized, targetType)
		} else {
			return BinaryProtoUnmarshal(serialized, targetType)
		}
	default:
		return fromBytes(serialized, target)
	}

}

func (g *StringSerializer) ToBytesFrom(entry interface{}) ([]byte, error) {
	switch v := entry.(type) {
	case string:
		return []byte(v), nil
	}
	return nil, ErrOnlyStringSupported
}

func (g *StringSerializer) FromBytesTo(serialized []byte, target interface{}) (interface{}, error) {
	switch t := target.(type) {
	case string:
		return string(serialized), nil
	default:
		return nil, fmt.Errorf(`type=%s: %w`, t, ErrOnlyStringSupported)
	}
}

//func (js *JSONSerializer) ToBytes(entry interface{}) ([]byte, error) {
//	return json.Marshal(entry)
//}
//
//func (js *JSONSerializer) FromBytes(serialized []byte, target interface{}) (interface{}, error) {
//	return JSONUnmarshalPtr(serialized, target)
//}

//func (ps *ProtoSerializer) ToBytes(entry interface{}) ([]byte, error) {
//	return proto.Marshal(entry.(proto.Message))
//}
//
//func (ps *ProtoSerializer) FromBytes(serialized []byte, target interface{}) (interface{}, error) {
//	return convert.FromBytes(serialized, target)
//}
