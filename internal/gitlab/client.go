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
	"golang.org/x/oauth2"
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

	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(ctx, src)

	var gl *gitlab.Client
	var err error

	if config.Av.GitLab.BaseURL == "" {
		gl, err = gitlab.NewClient(token)
	} else {
		gl, err = gitlab.NewClient(token, gitlab.WithBaseURL(config.Av.GitLab.BaseURL))
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to create GitLab client")
	}

	// Set custom HTTP client for timeout and auth
	gl.SetHTTPClient(httpClient)

	return &Client{httpClient, gl}, nil
}

func (c *Client) query(ctx context.Context, operation string, params interface{}) (reterr error) {
	log := logrus.WithFields(logrus.Fields{
		"operation": operation,
		"params":    logutils.Format("%#+v", params),
	})
	log.Debug("executing GitLab API query...")
	startTime := time.Now()
	defer func() {
		log := log.WithFields(logrus.Fields{
			"elapsed": time.Since(startTime),
		})
		if reterr != nil {
			log.WithError(reterr).Debug("GitLab API query failed")
		} else {
			log.Debug("GitLab API query succeeded")
		}
	}()
	
	// This is a helper method for logging consistency with GitHub client
	// Actual API calls will be made directly using the gitlab client methods
	return nil
}

func (c *Client) mutate(ctx context.Context, operation string, params interface{}) (reterr error) {
	log := logrus.WithFields(logrus.Fields{
		"operation": operation,
		"params":    logutils.Format("%#+v", params),
	})
	log.Debug("executing GitLab API mutation...")
	startTime := time.Now()
	defer func() {
		log := log.WithFields(logrus.Fields{
			"elapsed": time.Since(startTime),
		})
		if reterr != nil {
			log.WithError(reterr).Debug("GitLab API mutation failed")
		} else {
			log.Debug("GitLab API mutation succeeded")
		}
	}()
	
	// This is a helper method for logging consistency with GitHub client
	// Actual API calls will be made directly using the gitlab client methods
	return nil
}