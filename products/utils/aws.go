package utils

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/awslabs/aws-lambda-go-api-proxy/gorillamux"
)

type State struct {
	Adapter *gorillamux.GorillaMuxAdapterV2
}

func (s *State) LogRawRequestAndProxy(ctx context.Context, event events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	jRequest, _ := json.Marshal(event)
	log.Printf("Raw Input:\n %s\n", string(jRequest))
	resp, err := s.Adapter.Proxy(event)
	jResp, _ := json.Marshal(resp)
	log.Printf("Raw Output:\n %s\n", string(jResp))
	return resp, err
}
