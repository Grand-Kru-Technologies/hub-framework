package remoteforward

import (
	"errors"
	"testing"

	"github.com/QueerGlobal/hub-framework/util"
	"github.com/stretchr/testify/assert"
)

type recordingReporter struct {
	calls []string
}

func (r *recordingReporter) ReportError(err error, stage string) {
	r.calls = append(r.calls, stage+":"+err.Error())
}

func TestReportIfExhausted_ReportsRetryableFailure(t *testing.T) {
	reporter := &recordingReporter{}
	err := errors.New("all retry attempts failed: remote server returned status 503")

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
