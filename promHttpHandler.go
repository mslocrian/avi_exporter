package main

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	// "github.com/prometheus/common/log"
	"github.com/prometheus/common/expfmt"
)

//////////////////////////////////////////////////////////////////////////////////////////
// !WARNING! The following code was pulled from github.com/prometheus. It includes
// non-exportable functions that needed to be modified to support local
// queries of the Avi metric store on collect.
//////////////////////////////////////////////////////////////////////////////////////////
const (
	contentTypeHeader     = "Content-Type"
	contentLengthHeader   = "Content-Length"
	contentEncodingHeader = "Content-Encoding"
	acceptEncodingHeader  = "Accept-Encoding"
)

var (
	bufPool sync.Pool
)

func getBuf() *bytes.Buffer {
	buf := bufPool.Get()
	if buf == nil {
		return &bytes.Buffer{}
	}
	return buf.(*bytes.Buffer)
}

func giveBuf(buf *bytes.Buffer) {
	buf.Reset()
	bufPool.Put(buf)
}

func decorateWriter(request *http.Request, writer io.Writer, compressionDisabled bool) (io.Writer, string) {
	if compressionDisabled {
		return writer, ""
	}
	header := request.Header.Get(acceptEncodingHeader)
	parts := strings.Split(header, ",")
	for _, part := range parts {
		part := strings.TrimSpace(part)
		if part == "gzip" || strings.HasPrefix(part, "gzip;") {
			return gzip.NewWriter(writer), "gzip"
		}
	}
	return writer, ""
}

func aviPromHTTPHandler(e *Exporter, reg prometheus.Gatherer, opts promhttp.HandlerOpts) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		query := req.URL.Query()

		// Gather controller name from query string
		controller := query.Get("controller")
		if (len(query["controller"]) != 1) || (controller == "") {
			http.Error(w, "'controller' parameter must be specified once.", 400)
			return
		}

		// Gather tenant information from query string
		tenant := query.Get("tenant")
		if len(query["tenant"]) > 1 {
			http.Error(w, "'tenant' parameter can only be specified once.", 400)
			return
		}

		if tenant == "" {
			tenant = AVI_TENANT

		}

		// Gather AVI API Version from query string
		api_version := query.Get("api_version")
		if len(query["api_version"]) > 1 {
			http.Error(w, "'api_version' parameter can only be specified once.", 400)
			return
		}

		if api_version == "" {
			api_version = AVI_API_VERSION
		}

		err := e.Collect(controller, tenant, api_version)
		if err != nil {
			http.Error(w, fmt.Sprintf("encountered error in collection. error is: %s", err), 500)
			return
		}
		// START prometheus proprietary code.
		mfs, err := reg.Gather()
		if err != nil {
			if opts.ErrorLog != nil {
				opts.ErrorLog.Println("error gathering metrics:", err)
			}
			switch opts.ErrorHandling {
			case promhttp.PanicOnError:
				panic(err)
			case promhttp.ContinueOnError:
				if len(mfs) == 0 {
					http.Error(w, "No metrics gathered, last error:\n\n"+err.Error(), http.StatusInternalServerError)
					return
				}
			case promhttp.HTTPErrorOnError:
				http.Error(w, "An error has occurred during metrics gathering:\n\n"+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		contentType := expfmt.Negotiate(req.Header)
		buf := getBuf()
		defer giveBuf(buf)
		writer, encoding := decorateWriter(req, buf, opts.DisableCompression)
		enc := expfmt.NewEncoder(writer, contentType)
		var lastErr error
		for _, mf := range mfs {
			if err := enc.Encode(mf); err != nil {
				lastErr = err
				if opts.ErrorLog != nil {
					opts.ErrorLog.Println("error encoding metric family:", err)
				}
				switch opts.ErrorHandling {
				case promhttp.PanicOnError:
					panic(err)
				case promhttp.ContinueOnError:
					// Handled later.
				case promhttp.HTTPErrorOnError:
					http.Error(w, "An error has occurred during metrics encoding:\n\n"+err.Error(), http.StatusInternalServerError)
					return
				}
			}
		}
		if closer, ok := writer.(io.Closer); ok {
			closer.Close()
		}
		if lastErr != nil && buf.Len() == 0 {
			http.Error(w, "No metrics encoded, last error:\n\n"+err.Error(), http.StatusInternalServerError)
			return
		}
		header := w.Header()
		header.Set(contentTypeHeader, string(contentType))
		header.Set(contentLengthHeader, fmt.Sprint(buf.Len()))
		if encoding != "" {
			header.Set(contentEncodingHeader, encoding)
		}
		w.Write(buf.Bytes())
		// TODO(beorn7): Consider streaming serving of metrics.

	})
	// END prometheus proprietary code.
}
