//go:generate mockgen -build_flags=--mod=mod -package mock_client -destination=mock_client/client.go . Client

package client

import (
	"context"
	"encoding/json"

	ycsdk "github.com/yandex-cloud/go-sdk"
	"github.com/yandex-cloud/go-sdk/iamkey"
)

// YandexClient is a structure to access Yandex Cloud SDK.
type YandexClient struct {
	Compute
	ApplicationLoadBalancer
	sdk *ycsdk.SDK
}

// getClient returns authentificated YandexClient.
func getClient(ctx context.Context, key string) (Client, error) {
	var SAKey iamkey.Key

	if err := json.Unmarshal([]byte(key), &SAKey); err != nil {
		return nil, err
	}

	credentials, err := ycsdk.ServiceAccountKey(&SAKey)
	if err != nil {
		return nil, err
	}

	sdk, err := ycsdk.Build(ctx, ycsdk.Config{Credentials: credentials})
	if err != nil {
		return nil, err
	}

	return &YandexClient{sdk: sdk}, nil
}

// Close shutdowns Yandex SDK and closes all open connections.
func (c *YandexClient) Close(ctx context.Context) error {
	return c.sdk.Shutdown(ctx)
}
