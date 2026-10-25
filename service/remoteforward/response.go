package remoteforward

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/rs/zerolog"
)

type httpStatusClass int

const (
	statusClassSuccess httpStatusClass = iota
	statusClassClientError
	statusClassServerError
)

var statusClassHandlers = map[httpStatusClass]func(int, []byte, zerolog.Logger) (*entity.HttpServiceResponse, error){
	statusClassSuccess:     handleSuccessResponse,
	statusClassClientError: handleClientErrorResponse,
	statusClassServerError: handleServerErrorResponse,
}

func classifyHTTPStatus(statusCode int) httpStatusClass {
	switch {
	case statusCode >= 200 && statusCode < 300:
		return statusClassSuccess
	case statusCode >= 400 && statusCode < 500:
		return statusClassClientError
	default:
		return statusClassServerError
	}
}

// ReadAndParseResponse reads the HTTP response body once, classifies the result, and
// returns a ServiceResponse on success or a typed error on failure.
func ReadAndParseResponse(resp *http.Response, log zerolog.Logger) (*entity.HttpServiceResponse, error) {
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		wrapped := fmt.Errorf("%s: %w", util.UnrecoverableErrorMsg, err)
		log.Error().Err(wrapped).Str("stage", "read_response_body").Msg("remote forward read failed")
		return nil, wrapped
	}

	handler, ok := statusClassHandlers[classifyHTTPStatus(resp.StatusCode)]
	if !ok {
		return nil, fmt.Errorf("unexpected status class for code %d", resp.StatusCode)
	}

	parsed, err := handler(resp.StatusCode, body, log)
	if err != nil {
		return nil, err
	}

	return attachHTTPMeta(resp, parsed, body), nil
}

func attachHTTPMeta(resp *http.Response, parsed *entity.HttpServiceResponse, body []byte) *entity.HttpServiceResponse {
	if parsed != nil {
		return parsed
	}
	return &entity.HttpServiceResponse{
		ResponseMeta: &entity.HttpResponseMeta{
			OriginalResponse: resp,
			Status:           resp.Status,
			StatusCode:       resp.StatusCode,
			Proto:            resp.Proto,
			ProtoMajor:       resp.ProtoMajor,
			ProtoMinor:       resp.ProtoMinor,
			TransferEncoding: resp.TransferEncoding,
			Header:           resp.Header,
			Trailer:          resp.Trailer,
		},
		Body: body,
	}
}

func handleSuccessResponse(statusCode int, body []byte, log zerolog.Logger) (*entity.HttpServiceResponse, error) {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		log.Info().
			Int("status_code", statusCode).
			Str("stage", "parse_success_body").
			Msg("remote forward empty success body")
		return nil, nil
	}

	if !json.Valid(trimmed) {
		err := fmt.Errorf("%s: %w", util.UnrecoverableErrorMsg, ErrMalformedJSON)
		log.Error().
			Err(err).
			Int("status_code", statusCode).
			Int("body_bytes", len(body)).
			Str("stage", "parse_success_body").
			Msg("remote forward malformed JSON")
		return nil, err
	}

	var payload interface{}
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		err = fmt.Errorf("%s: %w", util.UnrecoverableErrorMsg, ErrMalformedJSON)
		log.Error().
			Err(err).
			Int("status_code", statusCode).
			Int("body_bytes", len(body)).
			Str("stage", "parse_success_body").
			Msg("remote forward malformed JSON")
		return nil, err
	}

	log.Info().
		Int("status_code", statusCode).
		Int("body_bytes", len(body)).
		Str("stage", "parse_success_body").
		Msg("remote forward JSON success")
	return nil, nil
}

func handleClientErrorResponse(statusCode int, body []byte, log zerolog.Logger) (*entity.HttpServiceResponse, error) {
	return nil, wrapUpstreamError(statusCode, body, log, true)
}

func handleServerErrorResponse(statusCode int, body []byte, log zerolog.Logger) (*entity.HttpServiceResponse, error) {
	return nil, wrapUpstreamError(statusCode, body, log, false)
}

func wrapUpstreamError(statusCode int, body []byte, log zerolog.Logger, unrecoverable bool) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return logUpstreamError(&UpstreamHTTPError{StatusCode: statusCode, Body: body, Payload: map[string]interface{}{}}, statusCode, len(body), log, unrecoverable)
	}

	if !json.Valid(trimmed) {
		return logUpstreamError(&NonJSONErrorResponse{StatusCode: statusCode, Body: body}, statusCode, len(body), log, unrecoverable)
	}

	var payload map[string]interface{}
	if err := json.Unmarshal(trimmed, &payload); err != nil {
		return logUpstreamError(&NonJSONErrorResponse{StatusCode: statusCode, Body: body}, statusCode, len(body), log, unrecoverable)
	}

	return logUpstreamError(&UpstreamHTTPError{StatusCode: statusCode, Body: body, Payload: payload}, statusCode, len(body), log, unrecoverable)
}

func logUpstreamError(cause error, statusCode, bodyBytes int, log zerolog.Logger, unrecoverable bool) error {
	var err error
	if unrecoverable {
		err = fmt.Errorf("%s: %w", util.UnrecoverableErrorMsg, cause)
	} else {
		err = cause
	}

	event := log.Warn().Err(err).Int("status_code", statusCode).Str("stage", "parse_error_body")
	if bodyBytes > 0 {
		event = event.Int("body_bytes", bodyBytes)
	}
	switch cause.(type) {
	case *NonJSONErrorResponse:
		event.Msg("remote forward non-JSON upstream error")
	default:
		event.Msg("remote forward upstream error")
	}
	return err
}

// ReportIfExhausted invokes report when retries are exhausted for a retryable failure.
func ReportIfExhausted(err error, report ErrorReporter) {
	if err == nil || report == nil {
		return
	}
	if strings.Contains(err.Error(), util.UnrecoverableErrorMsg) {
		return
	}
	if strings.Contains(err.Error(), "all retry attempts failed") {
		report.ReportError(err, "forward_exhausted")
	}
}
