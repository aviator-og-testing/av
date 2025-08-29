package gitlab

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"emperror.dev/errors"
	"github.com/aviator-co/av/internal/config"
	"github.com/aviator-co/av/internal/utils/logutils"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
)

type Client struct {
	httpClient *http.Client
	gl         *gitlab.Client
}

// NewClient creates a new GitLab client.
// It takes configuration from the global config.Av.GitLab variable.
func NewClient(ctx context.Context, token string) (*Client, error) {
	if token == "" {
		return nil, errors.Errorf("no GitLab token provided (do you need to configure one?)")
	}

	var gl *gitlab.Client
	var err error

	if config.Av.GitLab.BaseURL == "" {
		// Use GitLab.com
		gl, err = gitlab.NewClient(token)
	} else {
		// Use self-hosted GitLab instance
		gl, err = gitlab.NewClient(token, gitlab.WithBaseURL(config.Av.GitLab.BaseURL))
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to create GitLab client")
	}

	// Set timeout configuration
	gl.UserAgent = "aviator-cli"
	httpClient := gl.Client()
	httpClient.Timeout = 30 * time.Second

	return &Client{
		httpClient: httpClient,
		gl:         gl,
	}, nil
}

// query executes a GitLab API query (GET operation) with logging
func (c *Client) query(ctx context.Context, operation string, fn func() (interface{}, *gitlab.Response, error)) (result interface{}, reterr error) {
	log := logrus.WithFields(logrus.Fields{
		"operation": operation,
	})
	log.Debug("executing GitLab API query...")
	startTime := time.Now()
	defer func() {
		log := log.WithFields(logrus.Fields{
			"elapsed": time.Since(startTime),
			"result":  logutils.Format("%#+v", result),
		})
		if reterr != nil {
			log.WithError(reterr).Debug("GitLab API query failed")
		} else {
			log.Debug("GitLab API query succeeded")
		}
	}()

	result, resp, err := fn()
	if err != nil {
		return nil, c.wrapAPIError(err, resp)
	}
	return result, nil
}

// mutate executes a GitLab API mutation (POST/PUT/DELETE operation) with logging
func (c *Client) mutate(ctx context.Context, operation string, fn func() (interface{}, *gitlab.Response, error)) (result interface{}, reterr error) {
	log := logrus.WithFields(logrus.Fields{
		"operation": operation,
	})
	log.Debug("executing GitLab API mutation...")
	startTime := time.Now()
	defer func() {
		log := log.WithFields(logrus.Fields{
			"elapsed": time.Since(startTime),
			"result":  logutils.Format("%#+v", result),
		})
		if reterr != nil {
			log.WithError(reterr).Debug("GitLab API mutation failed")
		} else {
			log.Debug("GitLab API mutation succeeded")
		}
	}()

	result, resp, err := fn()
	if err != nil {
		return nil, c.wrapAPIError(err, resp)
	}
	return result, nil
}

// wrapAPIError wraps GitLab API errors with additional context
func (c *Client) wrapAPIError(err error, resp *gitlab.Response) error {
	if resp == nil {
		return errors.Wrap(err, "GitLab API error")
	}

	// Add HTTP status and URL information to the error
	baseErr := errors.Wrapf(err, "GitLab API error (status %d)", resp.StatusCode)
	
	if resp.Request != nil && resp.Request.URL != nil {
		// Redact sensitive information from URL
		safeURL := c.redactSensitiveURL(resp.Request.URL)
		return errors.Wrapf(baseErr, "request to %s", safeURL)
	}

	return baseErr
}

// redactSensitiveURL removes sensitive information from URLs for logging
func (c *Client) redactSensitiveURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	
	// Create a copy to avoid modifying the original
	safeCopy := *u
	
	// Remove query parameters that might contain sensitive information
	safeCopy.RawQuery = ""
	if safeCopy.User != nil {
		safeCopy.User = url.User("[redacted]")
	}
	
	return safeCopy.String()
}