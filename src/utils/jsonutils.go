// jsonutils
package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	simplejson "github.com/bitly/go-simplejson"
)

func ReadJson(filePath string) (*simplejson.Json, error) {
	bytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("%v\n", err)
		return simplejson.New(), err
	}
	result, err := simplejson.NewJson(bytes)

	if err != nil {
		fmt.Printf("%v\n", err)
		return simplejson.New(), err
	}
	return result, nil
}

func GetInt(Data map[string]interface{}, Key string) int64 {
	TempStr, ok := Data[Key]
	if !ok {
		return 0
	}
	sValue, ok1 := TempStr.(json.Number)
	if ok1 {
		result, _ := sValue.Int64()
		return result
	}

	sValue1, ok2 := TempStr.(int64)
	if ok2 {
		return sValue1
	}
	return 0
}

func GetString(Data map[string]interface{}, Key string) string {
	TempStr, ok := Data[Key]
	if !ok {
		return ""
	}
	return TempStr.(string)
}
