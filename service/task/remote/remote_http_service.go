package remote

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/service/remoteforward"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type ForwardingService struct {
	Host       string
	PathPrefix string
	name       string
	backoff    *util.Backoff
	client     *http.Client
	log        zerolog.Logger
	reporter   remoteforward.ErrorReporter
}

func NewForwardingService(config map[string]interface{}) (entity.Task, error) {
	svc := ForwardingService{}

	if host, ok := config["host"].(string); ok && host != "" {
		svc.Host = host
	}
	if svc.Host == "" {
		svc.Host = os.Getenv("PYTHON_HANDLER_URL")
	}

	if pathPrefix, ok := config["pathprefix"].(string); ok {
		svc.PathPrefix = remoteforward.NormalizePathPrefix(pathPrefix)
	}

	var backoffConfig util.BackoffConfig
	if backoff, ok := config["backoff"].(map[string]interface{}); ok {
		if initialDelay, ok := backoff["initialDelay"].(float64); ok {
			backoffConfig.InitialDelay = time.Duration(initialDelay) * time.Second
		}
		if maxDelay, ok := backoff["maxDelay"].(float64); ok {
			backoffConfig.MaxDelay = time.Duration(maxDelay) * time.Second
		}
		if multiplier, ok := backoff["multiplier"].(float64); ok {
			backoffConfig.Multiplier = multiplier
		}
		if maxRetries, ok := backoff["maxRetries"].(float64); ok {
			backoffConfig.MaxRetries = int(maxRetries)
		}
	}

	if backoffConfig.MaxRetries <= 0 {
		backoffConfig.MaxRetries = 3
	}
	if backoffConfig.InitialDelay <= 0 {
		backoffConfig.InitialDelay = 500 * time.Millisecond
	}
	if backoffConfig.MaxDelay <= 0 {
		backoffConfig.MaxDelay = 5 * time.Second
	}
	if backoffConfig.Multiplier <= 0 {
		backoffConfig.Multiplier = 2
	}

	backoff := util.NewBackoff(backoffConfig)
	svc.backoff = backoff

	opts := remoteforward.ServiceOptionsFromMap(config, "remote_task_forwarder")
	svc.client = &http.Client{Timeout: opts.ClientTimeout}
	svc.log = opts.Logger
	svc.reporter = opts.Reporter

	return &svc, nil
}

func (fs *ForwardingService) Apply(ctx context.Context, request entity.ServiceRequest) error {
	err := fs.backoff.ExecuteWithBackoff(func() error {
		response, err := fs.forwardRequest(ctx, request)
		if err != nil {
			return err
		}

		parsed, err := remoteforward.ReadAndParseResponse(response, fs.log)
		if err != nil {
			return err
		}

		request.SetResponse(parsed)
		return nil
	})
	if err != nil {
		remoteforward.ReportIfExhausted(err, fs.reporter)
		fs.log.Error().Err(err).Str("stage", "forward_apply").Msg("remote task forward failed")
		return err
	}

	return nil
}

func (fs *ForwardingService) Name() string {
	return fs.name
}

func (fs *ForwardingService) forwardRequest(ctx context.Context, request entity.ServiceRequest) (*http.Response, error) {
	jsonData, err := remoteforward.JSONPayloadForForward(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestPath := ""
	if request.GetURL() != nil {
		requestPath = request.GetURL().Path
	}

	urlString := remoteforward.JoinForwardURL(fs.Host, fs.PathPrefix, requestPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, urlString, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(req.Header))

	resp, err := fs.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			wrapped := fmt.Errorf("%s: %w", util.UnrecoverableErrorMsg, ctx.Err())
			fs.log.Warn().Err(wrapped).Str("stage", "http_do").Msg("remote task forward cancelled")
			return nil, wrapped
		}
		fs.log.Warn().Err(err).Str("stage", "http_do").Msg("remote task forward HTTP request failed")
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	return resp, nil
}
