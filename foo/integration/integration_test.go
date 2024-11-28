package integration

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cucumber/godog"
	"github.com/go-resty/resty/v2"
	log "github.com/sirupsen/logrus"
)

var baseUri = getBaseURI()
var ingressBaseUri = getIngressBaseUrl()
var request *resty.Request
var response resty.Response

func aRestService() {
	httpClient := resty.New()
	request = httpClient.R()
}

func iCallTheHelloWorldEndpoint() error {
	log.Printf("Hitting GET endpoint %s\n", baseUri)
	httpResponse, err := request.Get(baseUri + "/hello")

	if err != nil {
		return fmt.Errorf("call to %s was unsuccessful, error: %v", baseUri, err)
	}

	response = *httpResponse
	return nil
}

func iCallTheIngressHelloWorldEndpointAndWaitForItToBeReady() error {
	successful := false
	for i := 0; i < 8; i++ {
		log.Printf(" GET endpoint %s - retry number %d\n", ingressBaseUri, i)
		httpResponse, err := request.Get(ingressBaseUri + "/hello")
		if err != nil {
			log.Errorf("call to %s was unsuccessful, error: %v\n Sleeping for 10 seconds to wait for ingress to be available...", ingressBaseUri, err)
			time.Sleep(10 * time.Second)
			continue
		}
		response = *httpResponse
		successful = true
		break
	}

	if !successful {
		return fmt.Errorf("call to %s was unsuccessful, error: %v", ingressBaseUri, errors.New("unsuccessful call"))
	}

	return nil
}

func anOkResponseIsReturned() error {
	if response.IsSuccess() == true {
		return nil
	}
	return fmt.Errorf("response not successful, response code: %d, error: %v", response.StatusCode(), response.Error())
}

func InitializeScenario(ctx *godog.ScenarioContext) {
	ctx.Step(`^a rest service$`, aRestService)
	ctx.Step(`^an ok response is returned$`, anOkResponseIsReturned)
	ctx.Step(`^I call the hello world endpoint$`, iCallTheHelloWorldEndpoint)
	ctx.Step(`^I call the ingress hello world endpoint and wait for it to be ready$`, iCallTheIngressHelloWorldEndpointAndWaitForItToBeReady)
}

func getBaseURI() string {
	serviceEndpoint := os.Getenv("SERVICE_ENDPOINT")

	if serviceEndpoint == "" {
		return "http://service:8080"
	}
	return serviceEndpoint
}

func getIngressBaseUrl() string {
	return os.Getenv("INGRESS_ENDPOINT")
}
func TestFeatures(t *testing.T) {
	suite := godog.TestSuite{
		ScenarioInitializer: InitializeScenario,
		Options: &godog.Options{
			Format:   "pretty",
			Paths:    []string{"features"},
			TestingT: t,
		},
	}

	if suite.Run() != 0 {
		t.Fatal("non-zero status returned, failed to run feature tests")
	}
}