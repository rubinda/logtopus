<div align="center">
  <img src="docs/logtopus.png" width="250" alt="Logtopus logo" />
  <p align="center">
  <i>Capture and query service for semi structured data</i>
  </p>
</div>

# Logtopus

An application that can capture events and store them. Works with the standard [net/http](https://pkg.go.dev/net/http) library and [InfluxDB](https://www.influxdata.com/). Intended for deployments via [docker](https://docs.docker.com/get-docker/), more precisely [docker-compose](https://docs.docker.com/compose/).

## Installation

Clone the repository and run with docker compose:

```bash
git clone https://github.com/rubinda/logtopus.git
cd ./logtopus && docker compose up
```

## Usage

One can use `cURL` or your favourite API test tool (e.g [Insomnia](https://insomnia.rest/)). The API server listens on port 5000. All endpoints are prefixed with `/api/v1`.

### `/auth` <br>

issues tokens for authentication of other endpoints. Currently the user is hardcoded for test purposes.

```bash
curl --request POST --url http://localhost:5000/api/v1/auth --header 'Content-Type: application/json' --data '{"user":"johnnyHotbody","pass":"me-llamo-johnny"}'
```

### `/events` <br>

is a sink for storing information about events.

```bash
curl --request POST \
--url http://localhost:5000/api/v1/events \
--header 'Content-Type: application/json' \
--data '{
    "entityId": "plexServer001",
    "entityType": "mediaServer",
    "eventType": "downtime",
    "timestamp": "2023-02-05T19:43:06.159Z",
    "details": {
        "cause": "Planned maintenance",
        "severity": 4
    }
}'
```

The accepted JSON schema is as follows:
| field | type | |
| --- | --- | --- |
| entityId | string | required |
| eventType | string | required |
| entityType | string | optional |
| timestamp | string (RFC3339) | optional - server time used if not provided |
| details | object | optional - extra fields to store (with some limitations) |

### `/query/events` <br>

allows querying based on field values
Replace `<VALUE>` with actual token from the `auth/` endpoint. Data is a JSON object that contains conditions for returned objects. The `details` wrapper attribute is omitted for non-standard fields.

```bash
curl --request POST \
  --url http://localhost:5000/api/v1/query/events \
  --header 'Content-Type: application/json' \
  --header 'Token: <VALUE>' \
  --data '{
    "severity": 4,
    "eventType": "downtime"
  }'
```
