package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/rubinda/logtopus/pkg/http"
	"github.com/rubinda/logtopus/pkg/influxdb"
)

const (
	// apiServerURL defines the port the HTTP server listens on
	apiServerURL string = "0.0.0.0:5000"
)

func main() {
	// Optionally, an .env file can be given as the first parameter
	if len(os.Args) == 2 {
		envFilePath := os.Args[1]
		err := godotenv.Load(envFilePath)
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}
	influxURL := os.Getenv("INFLUXDB_HOST")
	influxOrg := os.Getenv("DOCKER_INFLUXDB_INIT_ORG")
	// TODO:
	//  - using admin token with full access,
	//    would be wiser to use custom acces management
	influxToken := os.Getenv("DOCKER_INFLUXDB_INIT_ADMIN_TOKEN")
	influxBucket := os.Getenv("DOCKER_INFLUXDB_INIT_BUCKET")
	jwtPrivateKeyPath := os.Getenv("JWT_PRIVATE_KEY")
	jwtPublicKeyPath := os.Getenv("JWT_PUBLIC_KEY")
	// TODO:
	//  - uses a self signed certificate, which causes warnings from clients (hence --insecure OR -k is needed for cURL)
	//	  for actual deployments something like Let's encrypt could be used (https://letsencrypt.org/)
	caCertFile := os.Getenv("SERVER_CERT_FILE")
	caKeyFile := os.Getenv("SERVER_KEY_FILE")

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
		DB:            influxClient,
		Address:       apiServerURL,
		CAKeyPath:     caKeyFile,
		CACertPath:    caCertFile,
		JWTKeyPath:    jwtPrivateKeyPath,
		JWTPubKeyPath: jwtPublicKeyPath,
	}
	http.ListenAndServe(httpServerConf)
}
