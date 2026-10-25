package remotehttpservice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/QueerGlobal/hub-framework/core/entity"
	"github.com/QueerGlobal/hub-framework/service/logging"
	"github.com/QueerGlobal/hub-framework/service/remoteforward"
	"github.com/QueerGlobal/hub-framework/util"
	"github.com/rs/zerolog"
)

type ForwardingService struct {
	Host       string
	PathPrefix string
	backoff    *util.Backoff
	client     *http.Client
	log        zerolog.Logger
	reporter   remoteforward.ErrorReporter
}

type ForwardingServiceConfig struct {
	Host          string
	PathPrefix    string
	BackoffConfig util.BackoffConfig
	ClientTimeout time.Duration
	Logger        *zerolog.Logger
	Reporter      remoteforward.ErrorReporter
}

func NewForwardingService(config interface{}) ForwardingService {
	svc := ForwardingService{}

	cfg := config.(ForwardingServiceConfig)

	svc.Host = cfg.Host
	svc.PathPrefix = normalizePathPrefix(cfg.PathPrefix)

	backoff := util.NewBackoff(cfg.BackoffConfig)
	svc.backoff = backoff

	timeout := cfg.ClientTimeout
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	svc.client = &http.Client{Timeout: timeout}

	if cfg.Logger != nil {
		svc.log = *cfg.Logger
	} else {
		svc.log = logging.GetLogger().With().Str("component", "remote_target_forwarder").Logger()
	}

	if cfg.Reporter != nil {
		svc.reporter = cfg.Reporter
	} else {
		svc.reporter = remoteforward.NoopErrorReporter()
	}

	return svc
}

func normalizePathPrefix(pathPrefix string) string {
	if len(pathPrefix) == 0 {
		return pathPrefix
	}
	if !strings.HasPrefix(pathPrefix, "/") {
		pathPrefix = "/" + pathPrefix
	}
	if !strings.HasSuffix(pathPrefix, "/") {
		pathPrefix = pathPrefix + "/"
	}
	return pathPrefix
}

func (fs *ForwardingService) Apply(request entity.ServiceRequest) (entity.ServiceResponse, error) {
	var serviceResponse entity.ServiceResponse

	err := fs.backoff.ExecuteWithBackoff(func() error {
		response, err := fs.forwardRequest(request)
		if err != nil {
			return err
		}

		parsed, err := remoteforward.ReadAndParseResponse(response, fs.log)
		if err != nil {
			return err
		}

		serviceResponse = parsed
		return nil
	})
	if err != nil {
		remoteforward.ReportIfExhausted(err, fs.reporter)
		return nil, err
	}

	return serviceResponse, nil
}

func (fs *ForwardingService) forwardRequest(request entity.ServiceRequest) (*http.Response, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	requestPath := ""
	if request.GetURL() != nil {
		requestPath = request.GetURL().Path
	}

	urlString := remoteforward.JoinForwardURL(fs.Host, fs.PathPrefix, requestPath)

	req, err := http.NewRequest(http.MethodPost, urlString, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := fs.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to perform HTTP request: %w", err)
	}

	return resp, nil
}
