package remoteforward

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

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

var statusClassHandlers = map[httpStatusClass]func(int, []byte, zerolog.Logger) error{
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

	handler := statusClassHandlers[classifyHTTPStatus(resp.StatusCode)]
	if err := handler(resp.StatusCode, body, log); err != nil {
		return nil, err
	}

	return buildHTTPServiceResponse(resp, body), nil
}

func buildHTTPServiceResponse(resp *http.Response, body []byte) *entity.HttpServiceResponse {
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

func handleSuccessResponse(statusCode int, body []byte, log zerolog.Logger) error {
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		log.Info().
			Int("status_code", statusCode).
			Str("stage", "parse_success_body").
			Msg("remote forward empty success body")
		return nil
	}

	if !json.Valid(trimmed) {
		err := fmt.Errorf("%s: %w", util.UnrecoverableErrorMsg, ErrMalformedJSON)
		log.Error().
			Err(err).
			Int("status_code", statusCode).
			Int("body_bytes", len(body)).
			Str("stage", "parse_success_body").
			Msg("remote forward malformed JSON")
		return err
	}

	log.Info().
		Int("status_code", statusCode).
		Int("body_bytes", len(body)).
		Str("stage", "parse_success_body").
		Msg("remote forward JSON success")
	return nil
}

func handleClientErrorResponse(statusCode int, body []byte, log zerolog.Logger) error {
	return wrapUpstreamError(statusCode, body, log, true)
}

func handleServerErrorResponse(statusCode int, body []byte, log zerolog.Logger) error {
	return wrapUpstreamError(statusCode, body, log, false)
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
