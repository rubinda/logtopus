package influxdb

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"github.com/rubinda/logtopus/pkg/parseutils"
)

const (
	// MeasurementFieldName is the JSON attribute name for InfluxDB _measurement field.
	MeasurementFieldName string = "entityType"
	// TimestampFieldName is the JSON attribute name for InfluxDB _time field.
	TimestampFieldName string = "timestamp"
)

var (
	// ErrFieldRequired is a message for missing required fields.
	ErrFieldRequired = fmt.Errorf("required field missing value")
	// hiddenFields are column names for InfluxDB fields which shouldn't be visible (as extra fields) to a regular client.
	hiddenFields = []string{"_measurement", "_start", "_stop", "_time", "result", "table"}
)

// ModelError represents an error with model structure.
type ModelError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// BasicEvent represents the data received by the API endpoints.
type BasicEvent struct {
	// EntityId describes a given event source (e.g. Customer ID).
	EntityId string `json:"entityId"`
	// EntityType represents type of the entity that produced the event (e.g. Customer, AutomatedTask, Admin).
	// Represents "_measurement" in InfluxDB
	EntityType string `json:"entityType"`
	// EventType is the (unique and descriptive) textual representation of occurred event (e.g. "account_creation", "customer_action", "billing")
	// Is part of InfluxDB tags
	EventType string `json:"eventType"`
	// Timestamp shoud be given in RFC3339 compatible format.
	// TODO:
	//  - using a single standard might be constrictive, based on use-cases it might be wise to change it for broader support
	Timestamp time.Time `json:"timestamp"`
	// EventDetails is an open map to provide values with some restrictions.
	// TODO:
	//  - for performance reasons with InfluxDB one shouldn't use the same key in fields as in measurements and/or tags.
	EventDetails map[string]any `json:"details"`
}

// QueryResultsToBasicEvents wraps values from InfluxDB table rows to a custom struct.
func QueryResultsToBasicEvents(result *api.QueryTableResult) (events []BasicEvent, err error) {
	events = make([]BasicEvent, 0)
	for result.Next() {
		values := result.Record().Values()
		event := BasicEvent{}
		event.EntityType = fmt.Sprint(parseutils.Pop(values, "_measurement"))
		event.EntityId = fmt.Sprint(parseutils.Pop(values, "entityId"))
		event.EventType = fmt.Sprint(parseutils.Pop(values, "eventType"))
		event.Timestamp = result.Record().Time()
		for _, key := range hiddenFields {
			delete(values, key)
		}
		event.EventDetails = values
		events = append(events, event)
	}
	err = result.Err()
	return
}

// ToPoint converts a JSON deserialized BasicEvent to a InfluxDB point ready to be written to the database.
func (e BasicEvent) ToPoint() (*write.Point, error) {
	// Parse extra fields (values which shouldn't be indexed in InfluxDB)
	extraFields := map[string]interface{}{
		"entityId": e.EntityId,
	}
	var err error
	// TODO:
	//  - based on more usecases, one could further improve the structuring capability (e.g. are nested objects needed, will there be complex lists, etc.)
	for key, value := range e.EventDetails {
		switch v := value.(type) {
		case bool:
			extraFields[key] = v
		case float64:
			// TODO:
			//  - if an int64 is given, it could exceed float space and thus be stored wrongly! -> are big numbers needed?
			if vAsInt := int(v); v == float64(vAsInt) {
				extraFields[key] = vAsInt
			} else {
				extraFields[key] = v
			}
		case string:
			extraFields[key] = v
		case []interface{}:
			extraFields[key] = make([]string, len(v))
			for i := range v {
				extraFields[key].([]string)[i] = fmt.Sprintf("%v", v[i])
			}
		case map[string]interface{}:
			extraFields[key], err = json.Marshal(v)
			if err != nil {
				return nil, err
			}
		case nil:
			log.Printf(`[WARNING] Skipping null field: "%s"`, key)
		default:
			log.Printf(`[WARNING] Unrecognized field structure in details given for field "%s" \n`, key)
		}
	}
	return influxdb2.NewPoint(
		e.EntityType,
		map[string]string{"eventType": e.EventType},
		extraFields,
		e.Timestamp,
	), nil
}

// Validate checks if all required fields have valid values. Returns a list of errors.
func (e *BasicEvent) Validate() []ModelError {
	problems := make([]ModelError, 0)
	if e.EntityId == "" {
		problems = append(problems, ModelError{"entityId", ErrFieldRequired.Error()})
	}
	if e.Timestamp.IsZero() {
		// TODO:
		//  - testing purposes, should be required!
		e.Timestamp = time.Now()
	}
	if e.EventType == "" {
		problems = append(problems, ModelError{"eventType", ErrFieldRequired.Error()})
	}
	return problems
}
