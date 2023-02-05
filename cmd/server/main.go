package main

import (
	"github.com/rubinda/logtopus/pkg/http"
	"github.com/rubinda/logtopus/pkg/influxdb"
)

const (
	influxURL         = "http://freya:8086/"
	influxOrg         = "Logtopus"
	influxBucket      = "auditLog"
	influxToken       = "ljZbsWKfbZJl-nNB8vNPzWXOZ0UhaH0jDLTwsL_lvHwHcMFccmONuGRoKhRZScZ7EnJjePe-DLMIJvcTPlSp6Q=="
	apiServerURL      = "localhost:5000"
	jwtPrivateKeyPath = "configs/jwtKey"
	jwtPublicKeyPath  = "configs/jwtKey.pub"
)

func main() {
	// Initialize a new authentication handler
	jwtAuth, err := http.NewJWTAuthority(jwtPrivateKeyPath, jwtPublicKeyPath)
	if err != nil {
		panic(err)
	}
	// Ensure a database client
	influxConf := influxdb.Configuration{
		ServerURL:    influxURL,
		Token:        influxToken,
		InfluxOrg:    influxOrg,
		InfluxBucket: influxBucket,
	}
	influxClient := influxdb.NewClient(influxConf)

	// Run the http(s) api server
	httpServerConf := http.Configuration{
		DB:      influxClient,
		Address: apiServerURL,
	}
	server := http.NewServer(httpServerConf, jwtAuth)
	server.Start()
}
