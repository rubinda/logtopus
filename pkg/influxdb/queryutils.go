package influxdb

import (
	"encoding/json"
	"fmt"
	"log"
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

// queryBuilder provides a way to achieve parametrised queries for InfluxDB OSS.
func queryBuilder(params map[string]any, bucket string) (query string) {
	startTime := parseutils.Pop(params, queryRangeStartTag)
	if startTime == nil {
		startTime = defaultQueryRangeStart
	}
	endTime := parseutils.Pop(params, queryRangeStopTag)
	if endTime == nil {
		endTime = time.Now().Format(time.RFC3339)
	}
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

	log.Print(query)
	return
}
