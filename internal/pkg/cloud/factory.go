package cloud

import (
	"context"

	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/gen/compute"
	"github.com/yandex-cloud/go-sdk/gen/loadbalancer"
)

type Cloud interface {
	Compute() *compute.Compute
	LoadBalancer() *loadbalancer.LoadBalancer
}

type Factory interface {
	NewCloud(context.Context) (Cloud, error)
}

type YandexFactory struct {
	Key string
}

func (ycf *YandexFactory) NewCloud(ctx context.Context) (Cloud, error) {
	sdk, err := ycsdk.Build(ctx, ycsdk.Config{
		Credentials: ycsdk.NewIAMTokenCredentials(ycf.Key),
	})

	return sdk, err
}
