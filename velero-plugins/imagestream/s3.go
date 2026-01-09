package imagestream

import (
	"context"

	"github.com/aws/aws-sdk-go/aws/endpoints"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

// GetBucketRegionFunc is the function used to get bucket region.
// It can be replaced in tests for mocking.
var GetBucketRegionFunc = getBucketRegionImpl

// GetBucketRegion returns the AWS region that a bucket is in, or an error
// if the region cannot be determined. This is a wrapper that calls GetBucketRegionFunc.
func GetBucketRegion(bucket string) (string, error) {
	return GetBucketRegionFunc(bucket)
}

func getBucketRegionImpl(bucket string) (string, error) {
	var region string

	session, err := session.NewSession()
	if err != nil {
		return "", errors.WithStack(err)
	}

	for _, partition := range endpoints.DefaultPartitions() {
		for regionHint := range partition.Regions() {
			region, _ = s3manager.GetBucketRegion(context.Background(), session, bucket, regionHint)

			// we only need to try a single region hint per partition, so break after the first
			break
		}

		if region != "" {
			return region, nil
		}
	}

	return "", errors.New("unable to determine bucket's region")
}
