package gaws

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

var notFoundError = gawsError{Type: "NotFound", Message: "Could not find something"}
var throttlingError = gawsError{Type: "Throttling", Message: "You have been throttled"}

func defaultRetryPredicate(status int, body []byte) (bool, error) {
	if status < 400 {
		return false, nil
	}

	// The request failed, but why?
	error := gawsError{}

	err := json.Unmarshal(body, &error)
	if err != nil {
		return false, err
	}

	// If the error wasn't about throttling and it is below 500, lets return it
	// This retries server errors or AWS errors where we should retry
	if error.Type != "Throttling" && status <= 500 {
		return false, error
	}

	return true, error
}

func testHTTP200(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("OK"))
}

func testHTTP404(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(notFoundError)

	w.WriteHeader(404)
	w.Write([]byte(b))
}

func testHTTP404NonJson(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(404)
	w.Write([]byte("I am not JSON!"))
}

func testAWSThrottle(w http.ResponseWriter, r *http.Request) {
	b, _ := json.Marshal(throttlingError)

	w.WriteHeader(400)
	w.Write([]byte(b))
}

func canonicalRequest() AWSRequest {
	r := AWSRequest{RetryPredicate: defaultRetryPredicate,
		Method:  "GET",
		Headers: map[string]string{}}
	return r
}

func TestSuccess(t *testing.T) {
	Convey("Given a request sent to a server that always returns 200s", t, func() {
		ts := httptest.NewServer(http.HandlerFunc(testHTTP200))
		defer ts.Close()

		r := canonicalRequest()
		r.URL = ts.URL

		_, err := r.Do()

		Convey("SendAWSRequest will not return errors", func() {
			So(err, ShouldBeNil)
		})

	})
}

func TestFailBadJson(t *testing.T) {
	Convey("Given a server that returns 404 errors without JSON", t, func() {

		ts := httptest.NewServer(http.HandlerFunc(testHTTP404NonJson))
		defer ts.Close()

		r := canonicalRequest()
		r.URL = ts.URL

		_, err := r.Do()

		Convey("SendAWSRequest should return an error", func() {
			So(err, ShouldNotBeNil)
		})

	})
}

func TestFailNoRetry(t *testing.T) {
	Convey("Given a server that returns 404 errors with proper JSON", t, func() {

		ts := httptest.NewServer(http.HandlerFunc(testHTTP404))
		defer ts.Close()

		r := canonicalRequest()
		r.URL = ts.URL

		_, err := r.Do()

		Convey("SendAWSRequest should return an error", func() {
			So(err, ShouldNotBeNil)
		})

		Convey("SendAWSRequest should return a not found error (and not attempt to retry)", func() {
			So(err.Error(), ShouldEqual, notFoundError.Error())
		})

	})
}

func TestThrottleRetry(t *testing.T) {
	Convey("Given a server that only returns 400 errors with the Trottle type", t, func() {

		ts := httptest.NewServer(http.HandlerFunc(testAWSThrottle))
		defer ts.Close()

		r := canonicalRequest()
		r.URL = ts.URL

		_, err := r.Do()

		Convey("SendAWSRequest should return an error", func() {
			So(err, ShouldNotBeNil)
		})

		Convey("SendAWSRequest should return an exceeded retries error", func() {
			So(err.Error(), ShouldEqual, exceededRetriesError.Error())
		})

	})
}

func TestGetRequest(t *testing.T) {

	Convey("When I use GetRequest", t, func() {
		r := canonicalRequest()
		r.URL = "http://www.google.com"
		r.Headers["foo"] = "bar"
		req := r.getRequest()

		Convey("It adds the headers", func() {
			So(req.Header["Foo"], ShouldResemble, []string{"bar"})
		})

		Convey("It sets the right method", func() {
			So(req.Method, ShouldEqual, "GET")
		})
	})
}

func TestBadRequest(t *testing.T) {

	Convey("When I send a request to a nonexistent host", t, func() {
		r := canonicalRequest()
		r.URL = "this will not work"
		_, err := r.Do()
		Convey("I get an error", func() {
			So(err, ShouldNotBeNil)
		})
	})
}
