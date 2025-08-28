package gitlab

import (
	"context"
	"net/http"
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

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	var gl *gitlab.Client
	var err error

	if config.Av.GitLab.BaseURL == "" {
		gl, err = gitlab.NewClient(token, gitlab.WithHTTPClient(httpClient))
	} else {
		gl, err = gitlab.NewClient(token, gitlab.WithBaseURL(config.Av.GitLab.BaseURL), gitlab.WithHTTPClient(httpClient))
	}

	if err != nil {
		return nil, errors.Wrap(err, "failed to create GitLab client")
	}

	return &Client{httpClient, gl}, nil
}

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
	
	result, _, err := fn()
	return result, err
}

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
	
	result, _, err := fn()
	return result, err
}