package remoteforward

import (
	"errors"
	"fmt"
	"testing"

	"github.com/QueerGlobal/hub-framework/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingReporter struct {
	calls []string
}

func (r *recordingReporter) ReportError(err error, stage string) {
	r.calls = append(r.calls, stage+":"+err.Error())
}

func TestReportIfExhausted_ReportsRetryableFailure(t *testing.T) {
	reporter := &recordingReporter{}
	err := fmt.Errorf("%w: %w", util.ErrAllRetriesFailed, errors.New("upstream 503"))

	ReportIfExhausted(err, reporter)

	assert.Len(t, reporter.calls, 1)
	assert.Contains(t, reporter.calls[0], "forward_exhausted")
}

func TestReportIfExhausted_SkipsUnrecoverable(t *testing.T) {
	reporter := &recordingReporter{}
	err := errors.New(util.UnrecoverableErrorMsg + ": bad request")

	ReportIfExhausted(err, reporter)

	assert.Empty(t, reporter.calls)
}

func TestReportIfExhausted_UsesSentinelNotMessageText(t *testing.T) {
	reporter := &recordingReporter{}
	err := errors.New("all retry attempts failed: something changed wording")

	ReportIfExhausted(err, reporter)

	assert.Empty(t, reporter.calls)
}

func TestRedactErrorForReport_StripsUpstreamBody(t *testing.T) {
	safe := redactErrorForReport(&UpstreamHTTPError{
		StatusCode: 400,
		Body:       []byte(`{"secret":"value"}`),
		Payload:    map[string]interface{}{"secret": "value"},
	})
	assert.NotContains(t, safe.Error(), "secret")
	assert.Contains(t, safe.Error(), "400")
}

func TestNewErrorReporter_NoopWhenDSNEmpty(t *testing.T) {
	r := NewErrorReporter("")
	_, ok := r.(noopErrorReporter)
	require.True(t, ok)
}
