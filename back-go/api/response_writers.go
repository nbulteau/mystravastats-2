package api

import (
	"bytes"
	"encoding"
	"encoding/json"
	"log"
	"math"
	"mystravastats/api/dto"
	"net/http"
	"reflect"
)

func writeJSON(writer http.ResponseWriter, status int, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(true)

	if err := enc.Encode(sanitizeJSONValue(v)); err != nil {
		return err
	}

	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_, _ = writer.Write(buf.Bytes())
	return nil
}

var (
	jsonMarshalerType = reflect.TypeOf((*json.Marshaler)(nil)).Elem()
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

func sanitizeJSONValue(value any) any {
	if value == nil {
		return nil
	}
	return sanitizeReflectValue(reflect.ValueOf(value)).Interface()
}

func sanitizeReflectValue(value reflect.Value) reflect.Value {
	if !value.IsValid() {
		return value
	}

	valueType := value.Type()
	if value.CanInterface() && (valueType.Implements(jsonMarshalerType) || valueType.Implements(textMarshalerType)) {
		return value
	}

	switch value.Kind() {
	case reflect.Interface:
		if value.IsNil() {
			return reflect.Zero(valueType)
		}
		sanitized := sanitizeReflectValue(value.Elem())
		if sanitized.Type().AssignableTo(valueType) {
			return sanitized
		}
		wrapped := reflect.New(valueType).Elem()
		wrapped.Set(sanitized)
		return wrapped
	case reflect.Pointer:
		if value.IsNil() {
			return reflect.Zero(valueType)
		}
		if value.CanInterface() && valueType.Implements(jsonMarshalerType) {
			return value
		}
		sanitized := sanitizeReflectValue(value.Elem())
		cloned := reflect.New(valueType.Elem())
		cloned.Elem().Set(sanitized)
		return cloned
	case reflect.Float32, reflect.Float64:
		if math.IsNaN(value.Float()) || math.IsInf(value.Float(), 0) {
			return reflect.Zero(valueType)
		}
		return value
	case reflect.Slice:
		if value.IsNil() {
			return reflect.MakeSlice(valueType, 0, 0)
		}
		cloned := reflect.MakeSlice(valueType, value.Len(), value.Len())
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(sanitizeReflectValue(value.Index(index)))
		}
		return cloned
	case reflect.Array:
		cloned := reflect.New(valueType).Elem()
		for index := 0; index < value.Len(); index++ {
			cloned.Index(index).Set(sanitizeReflectValue(value.Index(index)))
		}
		return cloned
	case reflect.Map:
		if value.IsNil() {
			return reflect.MakeMap(valueType)
		}
		cloned := reflect.MakeMapWithSize(valueType, value.Len())
		iter := value.MapRange()
		for iter.Next() {
			cloned.SetMapIndex(iter.Key(), sanitizeReflectValue(iter.Value()))
		}
		return cloned
	case reflect.Struct:
		cloned := reflect.New(valueType).Elem()
		cloned.Set(value)
		for index := 0; index < value.NumField(); index++ {
			field := valueType.Field(index)
			if field.PkgPath != "" {
				continue
			}
			cloned.Field(index).Set(sanitizeReflectValue(value.Field(index)))
		}
		return cloned
	default:
		return value
	}
}

func writeBadRequest(writer http.ResponseWriter, message string, description string) {
	writeAPIError(writer, http.StatusBadRequest, message, description)
}

func writeNotFound(writer http.ResponseWriter, message string, description string) {
	writeAPIError(writer, http.StatusNotFound, message, description)
}

func writeInternalServerError(writer http.ResponseWriter, description string) {
	writeAPIError(writer, http.StatusInternalServerError, "Internal server error", description)
}

func writeAPIError(writer http.ResponseWriter, status int, message string, description string) {
	if err := writeJSON(writer, status, dto.ErrorResponseMessageDto{
		Message:     message,
		Description: description,
		Code:        1,
	}); err != nil {
		log.Printf("failed to write API error response: %v", err)
	}
}
