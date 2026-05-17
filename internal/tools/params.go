package tools

import (
	"fmt"
	"strconv"
)

func getNumber(params map[string]any, key string) (float64, error) {
	val, ok := params[key]
	if !ok {
		return 0, fmt.Errorf("%s parameter is required", key)
	}

	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, fmt.Errorf("%s must be a number, got: %s", key, v)
		}
		return f, nil
	default:
		return 0, fmt.Errorf("%s must be a number, got: %T", key, val)
	}
}

func intParam(params map[string]any, key string, def, min, max int) int {
	if _, ok := params[key]; !ok {
		return def
	}
	n, err := getNumber(params, key)
	if err != nil {
		return def
	}
	v := int(n)
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
