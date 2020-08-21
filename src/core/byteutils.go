package core

import (
	"bytes"
	"encoding/binary"
	"math"
	"reflect"
	"runtime"
)

func Struct2Bytes(this reflect.Value) ([]byte, bool) {
	binData := bytes.NewBuffer([]byte{})
	v := this.Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() == reflect.Struct {
			bytes, ok := Struct2Bytes(f.Addr())
			if !ok {
				return nil, false
			}
			err := binary.Write(binData, binary.LittleEndian, bytes)
			if err != nil {
				return nil, false
			}
		} else if f.Kind() == reflect.String {
			strValue := f.Interface().(string)
			err := binary.Write(binData, binary.LittleEndian, []byte(strValue))
			if err != nil {
				LogError("StructToBytes string err. ", f.Kind())
				return nil, false
			}
		} else if f.Kind() == reflect.Bool {
			var err error
			if f.Interface().(bool) {
				err = binary.Write(binData, binary.LittleEndian, byte(1))
			} else {
				err = binary.Write(binData, binary.LittleEndian, byte(0))
			}

			if err != nil {
				LogError("StructToBytes bool err. ", f.Kind(), err)
				return nil, false
			}
		} else {
			err := binary.Write(binData, binary.LittleEndian, f.Interface())
			if err != nil {
				_, file, line, _ := runtime.Caller(2)
				LogError("StructToBytes others err. ", f.Kind(), err, " File", file, line)
				return nil, false
			}
		}
	}
	return binData.Bytes(), true
}

func Byte2Struct(dataType reflect.Value, bytes1 []byte) (bool, int) {
	v := dataType.Elem()
	index := 0
	numField := v.NumField()
	for i := 0; i < numField; i++ {
		if index >= len(bytes1) {
			break
		}
		f := v.Field(i)
		switch f.Kind() {
		case reflect.Bool:
			datas := bytes1[index]
			f.SetBool(datas != 0)
			index += 1
		case reflect.Int8:
			datas := bytes1[index]
			f.SetInt(int64(datas))
			index += 1
		case reflect.Int16:
			datas1 := bytes1[index : index+2]
			valueInt := int64(binary.LittleEndian.Uint16(datas1))
			f.SetInt(valueInt)
			index += 2
		case reflect.Int32:
			datas1 := bytes1[index : index+4]
			f.SetInt(int64(binary.LittleEndian.Uint32(datas1)))
			index += 4
		case reflect.Int:
			datas := bytes1[index : index+4]
			f.SetInt(int64(binary.LittleEndian.Uint32(datas)))
			index += 4
		case reflect.Int64:
			datas := bytes1[index : index+8]
			f.SetInt(int64(binary.LittleEndian.Uint64(datas)))
			index += 8
		case reflect.Uint8:
			datas := bytes1[index]
			f.SetUint(uint64(datas))
			index += 1
		case reflect.Uint16:
			datas1 := bytes1[index : index+2]
			f.SetUint(uint64(binary.LittleEndian.Uint16(datas1)))
			index += 2
		case reflect.Uint32:
			datas1 := bytes1[index : index+4]
			f.SetUint(uint64(binary.LittleEndian.Uint32(datas1)))
			index += 4
		case reflect.Uint64:
			datas := bytes1[index : index+8]
			f.SetUint(uint64(binary.LittleEndian.Uint64(datas)))
			index += 8
		case reflect.Float32:
			datas := bytes1[index : index+4]
			v := math.Float32frombits(binary.LittleEndian.Uint32(datas))
			f.SetFloat(float64(v))
			index += 4
		case reflect.Float64:
			datas := bytes1[index : index+8]
			f.SetFloat(float64(binary.LittleEndian.Uint64(datas)))
			index += 8
		case reflect.Array:
			cap := f.Cap()
			//如果是最后一个字段，可能实际发送的数据没有预计的长
			//使用有效数据
			if index+cap > len(bytes1) {
				cap = len(bytes1) - index
			}

			datas := bytes1[index : index+cap]
			CopyArray(f.Addr(), datas)
			index += cap
		case reflect.Slice:
			if i+1 != numField {
				LogError("BytesToStruct slice must be last element")
				return false, 0
			}
			temp := bytes1[index:]
			//			f.SetCap(len(temp))
			f.SetBytes(temp)
		case reflect.Struct:
			ok, len := Byte2Struct(f.Addr(), bytes1[index:])
			if !ok {
				return false, 0
			}
			index += len
		}
	}
	return true, index
}

func Byte2String(bytes []byte) string {
	var index int = len(bytes)
	var fIndex int = -1
	for i, v := range bytes {
		if v == 0 && fIndex != -1 {
			index = i
			break
		} else if fIndex == -1 {
			fIndex = i
		}
	}

	if fIndex == -1 {
		return ""
	}
	return string(bytes[fIndex:index])
}
