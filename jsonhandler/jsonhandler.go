package jsonhandler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

type JsonHandler struct {
	data []byte
}

func FromData(data []byte) (*JsonHandler, error) {
	if data == nil {
		return nil, errors.New("data is empty")
	}
	var j map[string]any
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, fmt.Errorf("data is not json \n\t%w", err)
	}
	return &JsonHandler{
		data: data,
	}, nil
}

func FromFile(path string) (*JsonHandler, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read json from '%s' error:\n\t%w", path, err)
	}
	return FromData(data)
}

func (j *JsonHandler) GetBool(path string) (bool, bool) {
	result := gjson.GetBytes(j.data, path)
	return result.Bool(), result.Exists()
}

func (j *JsonHandler) GetInt(path string) (int64, bool) {
	result := gjson.GetBytes(j.data, path)
	return result.Int(), result.Exists()
}

func (j *JsonHandler) GetResult(path string) (*gjson.Result, bool) {
	result := gjson.GetBytes(j.data, path)
	return &result, result.Exists()
}

func (j *JsonHandler) GetString(path string) (string, bool) {
	result := gjson.GetBytes(j.data, path)
	return result.String(), result.Exists()
}

func (j *JsonHandler) Set(path string, value any) error {
	newData, err := sjson.SetBytes(j.data, path, value)
	if err != nil {
		return fmt.Errorf("set '%s' value error:\n\t%w", path, err)
	}
	j.data = newData
	return nil
}

func (j *JsonHandler) SetRaw(path string, value []byte) error {
	newData, err := sjson.SetRawBytes(j.data, path, value)
	if err != nil {
		return fmt.Errorf("set '%s' value error:\n\t%w", path, err)
	}
	j.data = newData
	return nil
}

func (j *JsonHandler) Delete(path string) error {
	newData, err := sjson.DeleteBytes(j.data, path)
	if err != nil {
		return fmt.Errorf("delete '%s' value error:\n\t%w", path, err)
	}
	j.data = newData
	return nil
}

func (j *JsonHandler) Data() []byte {
	return j.data
}

func (j *JsonHandler) Format() error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, j.data, "", "  "); err != nil {
		return fmt.Errorf("format json error:\n\t%w", err)
	}
	j.data = buf.Bytes()
	return nil
}

func (j *JsonHandler) Compact() error {
	var buf bytes.Buffer
	if err := json.Compact(&buf, j.data); err != nil {
		return fmt.Errorf("compact json error:\n\t%w", err)
	}
	j.data = buf.Bytes()
	return nil
}

func (j *JsonHandler) SaveTo(destPath string) error {
	if err := os.WriteFile(destPath, j.data, 0660); err != nil {
		return fmt.Errorf("save json data to '%s' error:\n\t%w", destPath, err)
	}
	return nil
}
