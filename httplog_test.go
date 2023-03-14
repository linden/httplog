package httplog

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/exp/slog"
)

func parseLine(raw []byte) map[string]string {
	raw = raw[:len(raw)-2]

	result := make(map[string]string)

	var key []byte
	var value []byte

	inValue := false
	inString := false

	for _, character := range raw {
		switch {
		case character == '=' && inString == false:
			inValue = true

		case character == ' ' && inString == false:
			inValue = false

			result[string(key)] = string(value)

			key = []byte{}
			value = []byte{}

		case character == '"':
			if inString == true {
				inString = false
			} else {
				inString = true
			}

		default:
			if inValue == true {
				value = append(value, character)
			} else {
				key = append(key, character)
			}
		}
	}

	result[string(key)] = string(value)

	return result
}

func TestLogger(test *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/demo", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(200)
		writer.Header().Set("x-example", "example")
		writer.Write([]byte("Hello World"))
	})

	buffer := new(bytes.Buffer)
	logger := NewLogger(slog.New(slog.NewTextHandler(buffer)))

	server := httptest.NewServer(logger.Handler(mux))

	_, err := http.Post(server.URL+"/demo", "example/example", strings.NewReader("Hello World"))

	if err != nil {
		test.Fatalf("failed to make a request: %v\n", err)
	}

	raw := buffer.Bytes()
	line := parseLine(raw)

	test.Logf("line: %s\n", raw)

	if line["request-method"] != "POST" {
		test.Fatalf("expected request-method of GET got %s", line["request-method"])
	}

	if line["request-path"] != "/demo" {
		test.Fatalf("expected request-path of /demo got %s", line["request-path"])
	}

	if line["request-body"] != "Hello World" {
		test.Fatalf("expected request-body of Hello World got %s", line["request-body"])
	}

	if strings.Contains(line["request-headers"], "Content-Type:[example/example]") == false {
		test.Fatalf("expected request-headers to contain Content-Type:[example/example] got %s", line["request-headers"])
	}

	if line["response-status"] != "200" {
		test.Fatalf("expected request-path of 200 got %s", line["response-status"])
	}

	if line["response-body"] != "Hello World" {
		test.Fatalf("expected response-body of Hello World got %s", line["response-body"])
	}

	if strings.Contains(line["response-headers"], "X-Example:[example]") == false {
		test.Fatalf("expected response-headers to contain X-Example:[example] got %s", line["response-headers"])
	}

	if line["msg"] != "handled" {
		test.Fatalf("expected msg of handled got %s", line["msg"])
	}

	if line["level"] != "INFO" {
		test.Fatalf("expected level of INFO got %s", line["level"])
	}

	_, ok := line["time"]

	if ok == false {
		test.Fatal("missing time column")
	}
}

func ExampleLogger() {
	logger := NewLogger(slog.Default())

	mux := http.NewServeMux()

	mux.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.Write([]byte("Hello World"))
	})

	err := http.ListenAndServe(":8080", logger.Handler(mux))

	if err != nil {
		panic(err)
	}
}
