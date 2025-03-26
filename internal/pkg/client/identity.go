package client

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "sigs.k8s.io/controller-runtime/pkg/client"
)

// YandexClientProvider is a provider for YandexClient
type YandexClientProvider struct {
	// defaultKey is a default key for YandexClient from flags
	// for backward compatibility
	defaultKey string

	// controllerNamespace is a namespace where controller is running
	controllerNamespace string
}

// NewYandexClientProvider returns new ClientProvider
func NewYandexClientProvider(defaultKey, controllerNamespace string) *YandexClientProvider {
	return &YandexClientProvider{
		defaultKey:          defaultKey,
		controllerNamespace: controllerNamespace,
	}
}

// GetFromSecret returns YandexClient build from secret and key
func (b *YandexClientProvider) GetFromSecret(ctx context.Context, cl kube.Client, secretName, secretNamespace, keyName string) (Client, error) {
	secret := &corev1.Secret{}
	if err := cl.Get(ctx, kube.ObjectKey{Name: secretName, Namespace: secretNamespace}, secret); err != nil {
		return nil, err
	}

	key, ok := secret.Data[keyName]
	if !ok {
		return nil, fmt.Errorf("key %s not found in secret %s/%s", keyName, secretName, metav1.NamespaceDefault)
	}

	return getClient(ctx, string(key))
}

// GetDefault returns YandexClient build from default key
func (b *YandexClientProvider) GetDefault(ctx context.Context) (Client, error) {
	if b.defaultKey == "" {
		return nil, fmt.Errorf("default key is not set")
	}

	return getClient(context.Background(), b.defaultKey)
}

// GetFromKey returns YandexClient build from provided key
func (b *YandexClientProvider) GetFromKey(ctx context.Context, key string) (Client, error) {
	return getClient(ctx, key)
}
