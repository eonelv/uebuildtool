package sqlitedb

import (
	. "core"
)

/*
 * 数据库查询返回的行数据结果集
 */
type RowSet struct {
	Cols  []string
	Datas map[string][]byte
}

func (rowSet *RowSet) GetValue(name string, value interface{}) error {
	err := ConvertAssign(value, rowSet.Datas[name])
	if err != nil {
		return err
	}
	return nil
}

func (rowSet *RowSet) GetString(name string) string {
	var result string
	err := rowSet.GetValue(name, &result)
	if err != nil {
		LogError(err)
	}
	return result
}

func (rowSet *RowSet) GetObjectID(name string) ObjectID {
	return ObjectID(rowSet.GetUint64(name))
}
func (rowSet *RowSet) GetUint64(name string) uint64 {
	var result uint64
	err := rowSet.GetValue(name, &result)
	if err != nil {
		LogError(err)
		return 0
	}
	return result
}

func (rowSet *RowSet) GetInt(name string) int {
	var result int
	err := rowSet.GetValue(name, &result)
	if err != nil {
		LogError(err)
		return 0
	}
	return result
}

func (rowSet *RowSet) GetInt64(name string) int64 {
	var result int64
	err := rowSet.GetValue(name, &result)
	if err != nil {
		LogError(err)
		return 0
	}
	return result
}

func (rowSet *RowSet) GetFloat64(name string) float64 {
	var result float64
	err := rowSet.GetValue(name, &result)
	if err != nil {
		LogError(err)
		return 0
	}
	return result
}

func (rowSet *RowSet) GetBoolean(name string) bool {
	var result bool
	err := rowSet.GetValue(name, &result)
	if err != nil {
		LogError(err)
		return false
	}
	return result
}
