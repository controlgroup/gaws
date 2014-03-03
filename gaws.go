// Package gaws provides functions and variables that allow subpackages to work with AWS services.
package gaws

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"time"

	"github.com/smartystreets/go-aws-auth"
)

// MaxTries is the number of times to retry a failing AWS request.
var MaxTries int = 5

// AWSError is the error document returned from many AWS requests.
type AWSError struct {
	Type    string `json:"__type"`
	Message string `json:"message"`
}

var exceededRetriesError = AWSError{Type: "GawsExceededMaxRetries", Message: "The maximum number of retries for this request was exceeded."}

// Error formats the AWSError into an error message.
func (e AWSError) Error() string {
	return fmt.Sprintf("%v: %v", e.Type, e.Message)
}

// SendAWSRequest signs and sends an AWS request.
// It will retry 500s and throttling errors with an exponential backoff.
func SendAWSRequest(req *http.Request) ([]byte, error) {

	awsauth.Sign(req)
	client := &http.Client{}
	var lastBody []byte

	for try := 1; try < MaxTries; try++ {

		resp, err := client.Do(req)
		defer resp.Body.Close()

		if err != nil {
			return make([]byte, 0), err
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			return body, err
		}

		if resp.StatusCode < 400 {
			// The request succeeded
			return body, nil
		} else {

			// The request failed, but why?
			error := AWSError{}

			err = json.Unmarshal(body, &error)
			if err != nil {
				return body, err
			}

			// If the error wasn't about throttling and it is below 500, lets return it
			// This retries server errors or AWS errors where we should retry
			if error.Type != "Throttling" && resp.StatusCode <= 500 {
				return body, error
			}

			// Point lastBody to body
			lastBody = body

			// Exponential backoff for the retry
			sleepDuration := time.Duration(100 * math.Pow(2.0, float64(try)))
			time.Sleep(sleepDuration * time.Millisecond)
		}
	}
	return lastBody, exceededRetriesError
}
