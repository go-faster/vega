package cli

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/fatih/color"
	"github.com/go-faster/errors"
)

// Do is helper for Response that does not return the response.
func Do(ctx context.Context, client *http.Client, req *http.Request) error {
	res, err := Response(ctx, client, req)
	if err != nil {
		return errors.Wrap(err, "fail")
	}
	_ = res.Body.Close()
	return nil
}

// Response retries the request until it returns a response with status code 200.
//
// Status codes 400, 401, 422 are considered as permament errors.
func Response(ctx context.Context, client *http.Client, req *http.Request) (*http.Response, error) {
	var (
		start           = time.Now()
		constantBackoff = backoff.NewConstantBackOff(time.Second * 2)
		bo              = backoff.WithContext(constantBackoff, ctx)

		res        *http.Response
		lastStatus string
	)
	if err := backoff.RetryNotify(func() error {
		if req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return errors.Wrap(err, "get body")
			}
			req.Body = body
		}
		var doErr error
		if res, doErr = client.Do(req.WithContext(ctx)); doErr != nil {
			lastStatus = doErr.Error()
			return errors.Wrap(doErr, "do")
		}
		switch res.StatusCode {
		case http.StatusOK:
			return nil
		case http.StatusBadRequest, http.StatusUnauthorized:
			_ = res.Body.Close()
			return backoff.Permanent(errors.Errorf("status: %s", res.Status))
		default:
			_ = res.Body.Close()
			lastStatus = res.Status
			return errors.Errorf("status: %s", res.Status)
		}
	}, bo, func(err error, wait time.Duration) {
		fmt.Println(
			color.New(color.FgCyan).Sprintf("[%5s]", time.Since(start).Round(time.Second)),
			color.New().Sprint(req.Method),
			color.New(color.Faint).Sprint(req.URL),
			color.New(color.FgYellow).Sprint(lastStatus),
		)
	}); err != nil {
		return nil, errors.Wrap(err, "fail")
	}

	return res, nil
}
