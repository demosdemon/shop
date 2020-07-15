package awsparamstore

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ssm"

	"github.com/demosdemon/shop/pkg/secrets"
)

type resolver struct {
	client *ssm.SSM
	cache  map[string]string
}

func (r *resolver) Resolve(ctx context.Context, path string) (string, error) {
	if v, ok := r.cache[path]; ok {
		return v, nil
	}

	res, err := r.client.GetParameterWithContext(ctx, &ssm.GetParameterInput{
		Name:           aws.String(path),
		WithDecryption: aws.Bool(true),
	})

	if err != nil {
		if err, ok := err.(awserr.Error); ok {
			if err.Code() == ssm.ErrCodeParameterNotFound {
				return "", fmt.Errorf("no secret for path: %s", path)
			}
		}

		return "", err
	}

	v := aws.StringValue(res.Parameter.Value)
	r.cache[path] = v
	return v, nil
}

func New(config client.ConfigProvider) secrets.Resolver {
	r := &resolver{
		client: ssm.New(config, aws.NewConfig().WithRegion("us-west-2")),
		cache:  make(map[string]string),
	}
	return r
}
