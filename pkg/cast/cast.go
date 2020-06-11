package cast

import "github.com/spf13/cast"

func ToSliceE(i interface{}) ([]interface{}, error) {
	var s []interface{}

	switch v := i.(type) {
	case []string:
		for _, u := range v {
			s = append(s, u)
		}
		return s, nil
	}

	return cast.ToSliceE(i)
}

func ToSlice(i interface{}) []interface{} {
	v, _ := ToSliceE(i)
	return v
}
