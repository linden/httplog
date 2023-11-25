package httplog // import "github.com/linden/httplog"

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
)

type Logger struct {
	slog *slog.Logger
}

type ResponseWriter struct {
	Writer http.ResponseWriter

	status int
	body   *bytes.Buffer
}

func (w *ResponseWriter) Header() http.Header {
	// forward the underlying headers.
	return w.Writer.Header()
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	// write to the buffer.
	_, err := w.body.Write(p)
	if err != nil {
		return 0, err
	}

	// write to the connection.
	return w.Writer.Write(p)
}

func (w *ResponseWriter) WriteHeader(s int) {
	// set our status.
	w.status = s

	// forward the status.
	w.Writer.WriteHeader(s)
}

func (l *Logger) Handler(n http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// create a response writer.
		res := ResponseWriter{
			Writer: w,
			// create a buffer for
			body: new(bytes.Buffer),
		}

		// create a buffer for storing the request body.
		rb := new(bytes.Buffer)

		// check if we have a request body.
		if req.Body != nil {
			// create a tee reader, forwards any reads from the request body to our buffer.
			r := io.TeeReader(req.Body, rb)

			// set the request body to our tee reader.
			req.Body = io.NopCloser(r)
		}

		// forward the request to the next handler.
		n.ServeHTTP(&res, req)

		if req.Body != nil {
			// read the rest of the request body.
			io.ReadAll(req.Body)
		}

		// fallback level to info.
		lvl := slog.LevelInfo

		// check if the response status is in the 200 range.
		if res.status < 200 || res.status > 299 {
			// change the level to warn.
			lvl = slog.LevelWarn
		}

		// write the log message.
		l.slog.Log(
			context.Background(),
			lvl,
			"handled",
			"request-method",
			req.Method,
			"request-url",
			req.URL.String(),
			"request-headers",
			req.Header,
			"request-body",
			rb.String(),
			"response-status",
			res.status,
			"response-body",
			res.body.String(),
			"response-headers",
			res.Header(),
		)
	})
}

func NewLogger(sl *slog.Logger) Logger {
	return Logger{
		slog: sl,
	}
}
