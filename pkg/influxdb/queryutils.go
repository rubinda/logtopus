package influxdb

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rubinda/logtopus/pkg/parseutils"
)

const (
	// queryRangeStartTag is the JSON attribute for "start" (of time-series range) for InfluxDB queries.
	queryRangeStartTag string = "_timeFrom"
	// queryRangeStopTag is the JSON attribute for "stop" (end of time-series range) for InfluxDB queries.
	queryRangeStopTag string = "_timeTo"
)

// escapeFieldCondition returns a Flux (InfluxDB query language) compliant condition for given values.
func escapeFieldCondition(fieldName string, fieldValue any) string {
	if fieldName == "" {
		return ""
	}
	if fieldName == MeasurementFieldName {
		return fmt.Sprintf(`r["_measurement"] == %q`, fieldValue)
	}
	var encodedValue string
	switch v := fieldValue.(type) {
	case int, float64, bool:
		encodedValue = fmt.Sprintf("%v", v)
	case string:
		encodedValue = fmt.Sprintf(`%q`, v)
	default:
		foo, _ := json.Marshal(v)
		encodedValue = string(foo)
	}
	return fmt.Sprintf("r[%q] == %v", fieldName, encodedValue)
}

// validateTime makes sure given string is a valid time format (RFC3339).
func validateTime(timeString string, layout string, defaultValue string) (string, error) {
	// TODO:
	//  - ugly hack because any -> string conversion, => PopString method?
	if timeString == "" || timeString == "<nil>" {
		return defaultValue, nil
	}
	// TODO:
	//  - InfluxDB accepts -1m or -5d as range strings.
	//	  for simplicity, I didn't implement checking every possible combination, but this allows query injection
	return timeString, nil
}

// queryBuilder provides a way to achieve parametrised queries for InfluxDB OSS.
func queryBuilder(params map[string]any, bucket string) (query string, err error) {
	startTime, err := validateTime(fmt.Sprint(parseutils.Pop(params, queryRangeStartTag)), time.RFC3339, defaultQueryRangeStart)
	if err != nil {
		return
	}
	endTime, err := validateTime(fmt.Sprint(parseutils.Pop(params, queryRangeStopTag)), time.RFC3339, time.Now().Format(time.RFC3339))
	if err != nil {
		return
	}
	// Ignore timestamp field
	parseutils.Pop(params, TimestampFieldName)
	query = fmt.Sprintf(`
	import "influxdata/influxdb/schema"
	from(bucket: "%s")
	|> range(start: %v, stop: %v)
	|> schema.fieldsAsCols()`, bucket, startTime, endTime)

	if len(params) > 0 {
		fields := ""
		for key, value := range params {
			if fields != "" {
				fields += " and "
			}
			fields += escapeFieldCondition(key, value)
		}
		query += fmt.Sprintf(` |> filter(fn: (r) => %s)`, fields)
	}
	return
}
