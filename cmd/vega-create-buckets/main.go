package main

import (
	"context"
	"crypto/tls"
	"net/http"

	"github.com/go-faster/errors"
	"github.com/go-faster/sdk/app"
	"github.com/go-faster/sdk/zctx"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func main() {
	app.Run(func(ctx context.Context, lg *zap.Logger, m *app.Telemetry) error {
		const (
			endpoint        = "vega-hl.minio.svc.cluster.local:9000"
			accessKeyID     = "console"
			secretAccessKey = "console123"
		)
		client, err := minio.New(endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
			Secure: true,
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			},
		})
		if err != nil {
			return errors.Wrap(err, "minio.New")
		}
		for _, bucket := range []string{
			"loki-chunks",
			"loki-ruler",
			"loki-admin",
			"tempo",
		} {
			exists, err := client.BucketExists(ctx, bucket)
			if err != nil {
				return errors.Wrap(err, "BucketExists")
			}
			if exists {
				continue
			}
			if err := client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
				return errors.Wrap(err, "MakeBucket")
			}
		}
		zctx.From(ctx).Info("Created buckets")

		return nil
	},
		app.WithServiceName("vega-create-buckets"),
	)
}
