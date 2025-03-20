package sqlhelper

import (
	"mango/pkg/util/timehelper"
	"strconv"
	"time"
)

func ValueToInt(val interface{}, base int, bitSize int) int64 {
	if val == nil {
		return 0
	}
	switch val.(type) {
	case []byte:
		str := ByteToString(val.([]byte))
		u, _ := strconv.ParseInt(str, base, bitSize)
		return u
	default:
		return val.(int64)
	}
}

func ValueToUint(val interface{}, base int, bitSize int) uint64 {
	if val == nil {
		return 0
	}
	switch val.(type) {
	case []byte:
		str := ByteToString(val.([]byte))
		u, _ := strconv.ParseUint(str, base, bitSize)
		return u
	default:
		return uint64(val.(int64))
	}
}

func ValueToFloat(val interface{}, bitSize int) float64 {
	if val == nil {
		return 0
	}
	switch val.(type) {
	case []byte:
		f, _ := strconv.ParseFloat(ByteToString(val.([]byte)), bitSize)
		return f
	case int64:
		return float64(val.(int64))
	default:
		return val.(float64)
	}
}

type DataResult struct {
	RowCount int
	Rows     []interface{}
}

func (result *DataResult) GetRow(row int) []interface{} {
	return result.Rows[row].([]interface{})
}

func (result *DataResult) GetValue(row int, column int) interface{} {
	resultRow := result.GetRow(row)
	value := resultRow[column].(*interface{})
	return *value
}

func (result *DataResult) GetIntValue(row int, column int) int {
	val := result.GetValue(row, column)
	return int(ValueToInt(val, 10, 64))
}

func (result *DataResult) GetInt32Value(row int, column int) int32 {
	val := result.GetValue(row, column)
	return int32(ValueToInt(val, 10, 32))
}

func (result *DataResult) GetInt64Value(row int, column int) int64 {
	val := result.GetValue(row, column)
	return int64(ValueToInt(val, 10, 64))
}

func (result *DataResult) GetUIntValue(row int, column int) uint {
	val := result.GetValue(row, column)

	return uint(ValueToUint(val, 10, 64))
}

func (result *DataResult) GetUInt64Value(row int, column int) uint64 {
	val := result.GetValue(row, column)
	return ValueToUint(val, 10, 64)
}

func (result *DataResult) GetUInt32Value(row int, column int) uint32 {
	val := result.GetValue(row, column)

	return uint32(ValueToUint(val, 10, 32))
}

func (result *DataResult) GetUInt8Value(row int, column int) uint8 {
	val := result.GetValue(row, column)

	return uint8(ValueToUint(val, 10, 8))
}

func (result *DataResult) GetBoolValue(row int, column int) bool {
	val := result.GetValue(row, column)
	if val == nil {
		return false
	}
	switch val.(type) {
	case bool:
		return val.(bool)
	default:
		val = val.(int64)
		if val == int64(1) {
			return true
		}
	}
	return false
}

func (result *DataResult) GetFloat32Value(row int, column int) float32 {
	val := result.GetValue(row, column)
	return float32(ValueToFloat(val, 32))
}

func (result *DataResult) GetFloat64Value(row int, column int) float64 {
	val := result.GetValue(row, column)
	return ValueToFloat(val, 64)
}

func (result *DataResult) GetStringValue(row int, column int) string {
	val := result.GetValue(row, column)
	if val == nil {
		return ""
	}
	switch val.(type) {
	case []byte:
		return ByteToString(val.([]byte))
	default:
		return val.(string)
	}
}

func (result *DataResult) GetByteArrayValue(row int, column int) []byte {
	val := result.GetValue(row, column)
	if val == nil {
		return nil
	}

	switch val.(type) {
	case []byte:
		return val.([]byte)
	default:
		return []byte(val.(string))
	}
}

func (result *DataResult) GetTimeValue(row int, column int) *time.Time {
	val := result.GetValue(row, column)
	if val == nil {
		return nil
	}
	switch val.(type) {
	case []byte:
		timeStr := ByteToString(val.([]byte))
		t, _ := time.ParseInLocation(timehelper.Default, timeStr, time.Local) //string转time
		return &t
	default:
		return val.(*time.Time)
	}
}

func ByteToString(bs []byte) string {
	return string(bs)
}
