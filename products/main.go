package main

import (
	"context"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
	"github.com/gorilla/mux"

	"ms-astrid/products/middlewares"
	"ms-astrid/products/routes"
	"ms-astrid/products/utils"
)

func main() {
	utils.LoadEnv()

	client, err := utils.NewMongoClient(context.TODO())

	if err != nil {
		panic(err)
	}

	r := mux.NewRouter()
	r.Use(middlewares.CorsMiddleware)
	r.Use(middlewares.LoggerMiddleware(client))

	routes.RegisterRoutes(client, r)

	if os.Getenv("MODE") != "prod" {
		http.ListenAndServe(os.Getenv("SERVER_PORT"), r)
		return
	}

	adapter := gorillamux.NewV2(r)
	state := &utils.State{Adapter: adapter}
	lambda.Start(state.LogRawRequestAndProxy)
}
