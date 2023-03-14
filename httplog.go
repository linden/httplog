package httplog // import "github.com/linden/httplog"

import (
	"bytes"
	"io"
	"net/http"

	"golang.org/x/exp/slog"
)

type Logger struct {
	slogger *slog.Logger
}

type ResponseWriter struct {
	Writer http.ResponseWriter

	status int
	body   *bytes.Buffer
}

func (writer *ResponseWriter) Header() http.Header {
	return writer.Writer.Header()
}

func (writer *ResponseWriter) Write(raw []byte) (int, error) {
	_, err := writer.body.Write(raw)

	if err != nil {
		return 0, err
	}

	return writer.Writer.Write(raw)
}

func (writer *ResponseWriter) WriteHeader(status int) {
	writer.status = status
	writer.Writer.WriteHeader(status)
}

func (logger *Logger) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		response := ResponseWriter{
			Writer: writer,
			body:   new(bytes.Buffer),
		}

		requestBody := new(bytes.Buffer)

		if request.Body != nil {
			reader := io.TeeReader(request.Body, requestBody)

			request.Body = io.NopCloser(reader)
		}

		next.ServeHTTP(&response, request)

		if request.Body != nil {
			io.ReadAll(request.Body)
		}

		logger.slogger.Info(
			"handled",
			"request-method",
			request.Method,
			"request-path",
			request.URL.Path,
			"request-headers",
			request.Header,
			"request-body",
			requestBody.String(),
			"response-status",
			response.status,
			"response-body",
			response.body.String(),
			"response-headers",
			response.Header(),
		)
	})
}

func NewLogger(slogger *slog.Logger) Logger {
	return Logger{
		slogger: slogger,
	}
}
