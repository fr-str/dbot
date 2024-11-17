package dbot

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"unsafe"

	"github.com/bwmarrin/discordgo"
	"github.com/fr-str/log"
	"golang.org/x/exp/constraints"
)

var (
	prefix                             = "unmarshal form:"
	ErrUnmarshalFormUnsupported        = fmt.Errorf("%s unsupported type", prefix)
	ErrUnmarshalFormInvalidDest        = fmt.Errorf("%s dest is nil or not a pointer", prefix)
	ErrUnmarshalFormFailedToParseValue = fmt.Errorf("%s failed to parse value", prefix)
)

// Simple function to unmarshal form/query values to struct.
// Dest has to be a pointer.
// Supports:
//   - string
//   - all int, uint variants.
//   - int/string slices
func UnmarshalForm(options []*discordgo.ApplicationCommandInteractionDataOption, dest any) error {
	// check if dest is valid
	rv := reflect.ValueOf(dest)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return ErrUnmarshalFormInvalidDest
	}

	for i := range options {
		tagName := options[i].Name
		fieldValue := options[i]

		rv := reflect.ValueOf(dest).Elem()
		// find field for tag
		fieldIndex := findFieldByTag(rv, tagName)
		if fieldIndex == -1 {
			log.Debug("field not found for tag", log.String("tag", tagName))
			continue
		}

		// handle pointer
		field := rv.Field(fieldIndex)
		if field.Kind() == reflect.Pointer {
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}

			field = field.Elem()
		}

		// parse and set value
		switch field.Kind() {
		case reflect.Slice:
			sl := strings.Split(fieldValue.StringValue(), ",")
			err := fillSlice(field, sl)
			if err != nil {
				return err
			}

		case reflect.String:
			field.SetString(fieldValue.StringValue())

		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			field.SetInt(fieldValue.IntValue())

		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			field.SetUint(fieldValue.UintValue())
		}
	}

	return nil
}

// takes address of slice and puts data from 'fieldValue' into it
// handles parsing ~int
func fillSlice(rv reflect.Value, fieldValue []string) error {
	typ := rv.Type().Elem()
	switch typ.Kind() {
	case reflect.String:
		slPtr := (*[]string)(unsafe.Pointer(rv.UnsafeAddr()))
		*slPtr = fieldValue

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		slice := reflect.MakeSlice(rv.Type(), 0, len(fieldValue))
		for _, v := range fieldValue {
			i, err := strconv.Atoi(v)
			if err != nil {
				return err
			}

			x := reflect.ValueOf(i).Convert(typ)
			slice = reflect.Append(slice, x)
		}

		rv.Set(slice)
	}

	return nil
}

func findFieldByTag(rv reflect.Value, tagName string) int {
	for i := 0; i < rv.NumField(); i++ {
		field := rv.Type().Field(i)
		tag := field.Tag.Get("opt")
		if tag == tagName {
			return i
		}
	}
	return -1
}

// gets value from path value under given key and attempts to convert to given type
func GetPathValue[T constraints.Integer | string](r *http.Request, key string) (T, error) {
	val := r.PathValue(key)
	return CastAs[T](val)
}

// gets value from path value under given key and attempts to convert to given type
// returns no error if value under key is empty string
func GetFormValue[T constraints.Integer | string](r *http.Request, key string) (T, error) {
	var def T
	val := r.FormValue(key)
	if val == "" {
		return def, nil
	}

	return CastAs[T](val)
}

func CastAs[T constraints.Integer | string](val string) (T, error) {
	var t T
	var ret any
	var err error
	switch any(t).(type) {
	case string:
		return any(val).(T), nil

	case uint, uint8, uint16, uint32, uint64:
		// nolint:gosec
		ret, err = strconv.ParseUint(val, 10, int(unsafe.Sizeof(t))*8)

	case int, int8, int16, int32, int64:
		// nolint:gosec
		ret, err = strconv.ParseInt(val, 10, int(unsafe.Sizeof(t))*8)
	}
	if err != nil {
		return t, err
	}

	ret = reflect.ValueOf(ret).Convert(reflect.TypeOf(t)).Interface()

	return ret.(T), nil
}
