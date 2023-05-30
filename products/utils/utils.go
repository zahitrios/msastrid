package utils

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/newrelic/go-agent/v3/newrelic"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetReader(filename string) (*csv.Reader, error) {
	res, err := MakeRequest(filename, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	reader := res.Body
	csvReader := csv.NewReader(reader)

	if _, err := csvReader.Read(); err != nil {
		return nil, err
	}

	return csvReader, nil
}

func LoadEnv() {
	pwd, _ := os.Getwd()
	env := fmt.Sprintf("%s/products/local.env", pwd)
	godotenv.Load(env)
}

func MakeRequest(url string, method string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("making request: %s", err.Error())
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("response: %s", err.Error())
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("response is no success: %s", err.Error())
	}

	return response, nil
}

func NewMongoClient(ctx context.Context) (*mongo.Client, error) {
	serverAPIOptions := options.ServerAPI(options.ServerAPIVersion1)
	clientOptions := options.Client().
		ApplyURI(os.Getenv("MONGO_URI")).
		SetServerAPIOptions(serverAPIOptions)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return mongo.Connect(ctx, clientOptions)
}

func NewRelicApp() (*newrelic.Application, error) {
	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName(os.Getenv("NEW_RELIC_APP_NAME")),
		newrelic.ConfigLicense(os.Getenv("NEW_RELIC_LICENSE")),
		newrelic.ConfigAppLogForwardingEnabled(true),
	)

	if err != nil {
		fmt.Println("unable to create New Relic Application", err)
	}

	return app, err
}
