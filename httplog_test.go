package httplog

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func parseLine(p []byte) map[string]string {
	// trim the line delimiter from the end.
	p = p[:len(p)-2]

	// create new map to store the components of the line.
	r := make(map[string]string)

	var k []byte
	var v []byte

	inV := false
	inS := false

	// iterate over remaining every charecter in the line.
	for _, c := range p {
		switch {
		// check if we've moved from the key to the value portion of the component.
		case c == '=' && inS == false:
			inV = true

		// check if we're starting a new key component.
		case c == ' ' && inS == false:
			inV = false

			// store the complete component in the map.
			r[string(k)] = string(v)

			// create new empty key and value.
			k = []byte{}
			v = []byte{}

		// check if we're starting or ending a string.
		case c == '"':
			if inS == true {
				inS = false
			} else {
				inS = true
			}

		// add to either the key of the value.
		default:
			if inV == true {
				v = append(v, c)
			} else {
				k = append(k, c)
			}
		}
	}

	// add the last key and value.
	r[string(k)] = string(v)

	return r
}

func TestLogger(test *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/demo", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
		writer.Header().Set("x-example", "example")
		writer.Write([]byte("Hello World"))
	})

	// create new buffer for storing the logs.
	b := new(bytes.Buffer)

	// create new logger with a text-formatted handler.
	hl := NewLogger(slog.New(slog.NewTextHandler(b, nil)))

	// create new testing server.
	s := httptest.NewServer(hl.Handler(mux))

	// send a request to our testing server.
	_, err := http.Post(s.URL+"/demo", "example/example", strings.NewReader("Hello World"))
	if err != nil {
		test.Fatalf("failed to make a request: %v\n", err)
	}

	// convert the buffer into a slice.
	r := b.Bytes()

	// parse the line.
	line := parseLine(r)

	test.Logf("line: %s\n", r)

	// ensure the request method matches.
	if line["request-method"] != "POST" {
		test.Fatalf("expected request-method of GET got %s", line["request-method"])
	}

	// ensure the request URL matches.
	if line["request-url"] != "/demo" {
		test.Fatalf("expected request-url of /demo got %s", line["request-url"])
	}

	// ensure the request body matches.
	if line["request-body"] != "Hello World" {
		test.Fatalf("expected request-body of Hello World got %s", line["request-body"])
	}

	// ensure the request headers contain a content-type header that matches what we expect.
	if strings.Contains(line["request-headers"], "Content-Type:[example/example]") == false {
		test.Fatalf("expected request-headers to contain Content-Type:[example/example] got %s", line["request-headers"])
	}

	// ensure the response status matches.
	if line["response-status"] != "200" {
		test.Fatalf("expected request-path of 200 got %s", line["response-status"])
	}

	// ensure the response body matches.
	if line["response-body"] != "Hello World" {
		test.Fatalf("expected response-body of Hello World got %s", line["response-body"])
	}

	// ensure the response header contain the "X-Example" header.
	if strings.Contains(line["response-headers"], "X-Example:[example]") == false {
		test.Fatalf("expected response-headers to contain X-Example:[example] got %s", line["response-headers"])
	}

	// ensure the message is "handled".
	if line["msg"] != "handled" {
		test.Fatalf("expected msg of handled got %s", line["msg"])
	}

	// ensure the level is "INFO".
	if line["level"] != "INFO" {
		test.Fatalf("expected level of INFO got %s", line["level"])
	}

	// ensure we have a time column.
	_, ok := line["time"]
	if ok == false {
		test.Fatal("missing time column")
	}
}

func ExampleLogger() {
	// create new logger using the default `slog.Logger`.
	hl := NewLogger(slog.Default())

	// create new mux so we can easily forward it with the middleware.
	mux := http.NewServeMux()

	// handle all requests with a "Hello World" response.
	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("Hello World"))
	})

	// listen and serve using the logger as a middleware.
	err := http.ListenAndServe(":8080", hl.Handler(mux))
	if err != nil {
		panic(err)
	}
}
