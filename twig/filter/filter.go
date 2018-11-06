// Package filter provides built-in filters for Twig-compatibility.
package filter

import (
	"fmt"
	"math"
	"net/url"
	"strings"
	"unicode/utf8"

	"reflect"
	"time"

	"github.com/polakto/stick"
)

const (
	FilterDateDefaultLayout     = "yyyy-MM-dd"
	FilterDateTimeDefaultLayout = "yyyy-MM-dd hh:mm:ss"
	FilterTimeDefaultLayout     = "hh:mm:ss"
)

// by polakto
var DatePatternTokensMap = map[string]string{
	// year
	"yyyy": "2006",
	"yyy":  "006",
	"yy":   "06",

	// month
	"MM": "01", // 01-12
	"M":  "1",  // 1-12

	// day
	"dd": "02", // 01-07
	"d":  "2",  // 1-7

	// hours
	"hh": "03", // 01-12
	"h":  "3",  // 1-12
	"HH": "15", // 0 - 24
	"H":  "15", // 0 - 24 - not possible, will be 00 - 24

	// minutes
	"mm": "04", // 00 - 59
	"m":  "4",  // 0 - 59

	// seconds
	"ss": "05", // 00-59
	"s":  "5",  // 0-59
}

var DatePatternTokensSlice = []string{
	"yyyy",
	"yyy",
	"yy",
	"MM",
	"M",
	"dd",
	"d",
	"hh",
	"h",
	"HH",
	"H",
	"mm",
	"m",
	"ss",
	"s",
}

// by polakto
func StandardDatePatternToGoDatePattern(stdPattern string) string {
	goPattern := stdPattern
	for key := range DatePatternTokensSlice {
		goPattern = strings.Replace(goPattern, DatePatternTokensSlice[key], DatePatternTokensMap[DatePatternTokensSlice[key]], -1)

	}
	return goPattern
}

// builtInFilters returns a map containing all built-in Twig filters,
// with the exception of "escape", which is provided by the AutoEscapeExtension.
func TwigFilters() map[string]stick.Filter {
	return map[string]stick.Filter{
		"abs":              filterAbs,
		"default":          filterDefault,
		"batch":            filterBatch,
		"capitalize":       filterCapitalize,
		"convert_encoding": filterConvertEncoding,
		"date":             filterDate,
		"date_modify":      filterDateModify,
		"first":            filterFirst,
		"format":           filterFormat,
		"join":             filterJoin,
		"json_encode":      filterJSONEncode,
		"keys":             filterKeys,
		"last":             filterLast,
		"length":           filterLength,
		"lower":            filterLower,
		"merge":            filterMerge,
		"nl2br":            filterNL2BR,
		"number_format":    filterNumberFormat,
		"raw":              filterRaw,
		"replace":          filterReplace,
		"reverse":          filterReverse,
		"round":            filterRound,
		"slice":            filterSlice,
		"sort":             filterSort,
		"split":            filterSplit,
		"striptags":        filterStripTags,
		"title":            filterTitle,
		"trim":             filterTrim,
		"upper":            filterUpper,
		"url_encode":       filterURLEncode,

		// custom
		"get":      filterGet,
		"dateTime": filterDateTime,
		"time":     filterTime,
	}
}

// filterAbs takes no arguments and returns the absolute value of val.
// Value val will be coerced into a number.
func filterAbs(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	n := stick.CoerceNumber(val)
	if 0 == n {
		return n
	}
	return math.Abs(n)
}

// filterBatch takes 2 arguments and returns a batched version of val.
// Value val must be a map, slice, or array. The filter has two optional arguments: number
// of items per batch (defaults to 1), and the default fill value. If the
// fill value is not specified, the last group of batched values may be smaller than
// the number specified as items per batch.
func filterBatch(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	perSlice := 1
	var blankValue stick.Value
	if l := len(args); l >= 1 {
		perSlice = int(stick.CoerceNumber(args[0]))
		if l >= 2 {
			blankValue = args[1]
		}
	}
	if !stick.IsIterable(val) {
		// TODO: This would trigger an E_WARNING in PHP.
		return nil
	}
	if perSlice <= 1 {
		// TODO: This would trigger an E_WARNING in PHP.
		return nil
	}
	l, _ := stick.Len(val)
	numSlices := int(math.Ceil(float64(l) / float64(perSlice)))
	out := make([][]stick.Value, numSlices)
	curr := []stick.Value{}
	i := 0
	j := 0
	_, err := stick.Iterate(val, func(k, v stick.Value, l stick.Loop) (bool, error) {
		// Use a variable length slice and append(). This maintains
		// correct compatibility with Twig when the fill value is nil.
		curr = append(curr, v)
		j++
		if j == perSlice {
			out[i] = curr
			curr = []stick.Value{}
			i++
			j = 0
		}
		return false, nil
	})
	if err != nil {
		// TODO: Report error
		return nil
	}
	if i != numSlices {
		for ; blankValue != nil && j < perSlice; j++ {
			curr = append(curr, blankValue)
		}
		out[i] = curr
	}
	return out
}

// filterCapitalize takes no arguments and returns val with the first
// character capitalized.
func filterCapitalize(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	s := stick.CoerceString(val)
	return strings.ToUpper(s[:1]) + s[1:]
}

func filterConvertEncoding(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterDate(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	requestedLayout := FilterDateDefaultLayout

	strD := stick.CoerceString(val)
	d, conversionErr := convertMariaDBDate(strD)
	if conversionErr != nil {
		return nil
	}

	if l := len(args); l >= 1 {
		requestedLayout = stick.CoerceString(args[0])
	}

	return fmt.Sprintf("\n %s", d.Format(StandardDatePatternToGoDatePattern(requestedLayout)))
}

func filterDateTime(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	requestedLayout := FilterDateTimeDefaultLayout

	strD := stick.CoerceString(val)
	d, conversionErr := convertMariaDBDateTime(strD)
	if conversionErr != nil {
		return nil
	}

	if l := len(args); l >= 1 {
		requestedLayout = stick.CoerceString(args[0])
	}

	return fmt.Sprintf("\n %s", d.Format(StandardDatePatternToGoDatePattern(requestedLayout)))
}

func filterTime(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	requestedLayout := FilterTimeDefaultLayout

	strD := stick.CoerceString(val)
	d, conversionErr := convertMariaDBTime(strD)
	if conversionErr != nil {
		return nil
	}

	if l := len(args); l >= 1 {
		requestedLayout = stick.CoerceString(args[0])
	}

	return fmt.Sprintf("\n %s", d.Format(StandardDatePatternToGoDatePattern(requestedLayout)))
}

// filter date, time, datetime helpers
func convertMariaDBDate(in string) (time.Time, error) {
	t, err := time.Parse("2006-01-02", in)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func convertMariaDBTime(in string) (time.Time, error) {
	t, err := time.Parse("15:04:05", in)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func convertMariaDBDateTime(in string) (time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", in)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

func filterDateModify(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

// filterDefault takes one argument, the default value. If val is empty,
// the default value will be returned.
func filterDefault(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	var d stick.Value
	if len(args) > 0 {
		d = args[0]
	}
	if stick.CoerceString(val) == "" {
		return d
	}
	return val
}

func filterFirst(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	if stick.IsArray(val) {
		arr := reflect.ValueOf(val)
		return arr.Index(0).Interface()
	}

	if stick.IsMap(val) {
		// TODO: Trigger runtime error, Golang randomises map keys so getting the "First" does not make sense
		return nil
	}

	if s := stick.CoerceString(val); s != "" {
		return string(s[0])
	}

	return nil
}

func filterFormat(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterJoin(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	if !stick.IsIterable(val) {
		return nil
	}

	separator := ``
	if len(args) == 1 {
		separator = stick.CoerceString(args[0])
	}

	var slice []string
	stick.Iterate(val, func(k, v stick.Value, l stick.Loop) (bool, error) {
		slice = append(slice, stick.CoerceString(v))
		return false, nil
	})

	return strings.Join(slice, separator)
}

func filterJSONEncode(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterKeys(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterLast(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

// filterLength returns the length of val.
func filterLength(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	if v, ok := val.(string); ok {
		return utf8.RuneCountInString(v)
	}
	l, _ := stick.Len(val)
	// TODO: Report error
	return l
}

// filterLower returns val transformed to lower-case.
func filterLower(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return strings.ToLower(stick.CoerceString(val))
}

func filterMerge(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	if !stick.IsIterable(val) {
		return nil
	}

	if len(args) != 1 {
		return nil
	}

	var out []stick.Value

	stick.Iterate(val, func(k, v stick.Value, l stick.Loop) (bool, error) {
		out = append(out, v)
		return false, nil
	})

	stick.Iterate(args[0], func(k, v stick.Value, l stick.Loop) (bool, error) {
		out = append(out, v)
		return false, nil
	})

	return out
}

func filterNL2BR(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterNumberFormat(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterRaw(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterReplace(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterReverse(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterRound(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterSlice(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterSort(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterSplit(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

func filterStripTags(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// TODO: Implement Me
	return val
}

// filterTitle returns val with the first character of each word capitalized.
func filterTitle(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return strings.Title(stick.CoerceString(val))
}

// filterTrim returns val with whitespace trimmed on both left and ride sides.
func filterTrim(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return strings.TrimSpace(stick.CoerceString(val))
}

// filterUpper returns val in upper-case.
func filterUpper(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	return strings.ToUpper(stick.CoerceString(val))
}

func filterURLEncode(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// by polakto
	return url.PathEscape(stick.CoerceString(val))
}

func filterGet(ctx stick.Context, val stick.Value, args ...stick.Value) stick.Value {
	// by polakto
	var intKey int
	var strKey string

	if len(args) != 1 {
		return nil
	}
	switch args[0].(type) {
	case string:
		strKey = args[0].(string)
	case float64:
		intKey = int(args[0].(float64))
	default:
		return nil
	}

	switch val.(type) {
	case []stick.Value:
		if intKey == 0 {
			return nil
		}
		if intKey > len(val.([]stick.Value)) {
			return nil
		}
		return val.([]stick.Value)[intKey-1]
	case map[string]stick.Value:
		if strKey == "" {
			return nil
		}
		mapData := val.(map[string]stick.Value)
		item, ok := mapData[strKey]
		if !ok {
			return nil
		}
		return item
	default:
		return nil
	}
}
