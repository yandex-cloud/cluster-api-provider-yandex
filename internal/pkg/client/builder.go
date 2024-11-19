package client

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kube "sigs.k8s.io/controller-runtime/pkg/client"
)

// YandexClientBuilder is a builder for YandexClient
type YandexClientBuilder struct {
	// defaultKey is a default key for YandexClient from flags
	// for backward compatibility
	defaultKey string

	// controllerNamespace is a namespace where controller is running
	controllerNamespace string
}

// NewYandexClientBuilder returns new YandexClientBuilder
func NewYandexClientBuilder(defaultKey, controllerNamespace string) *YandexClientBuilder {
	return &YandexClientBuilder{
		defaultKey:          defaultKey,
		controllerNamespace: controllerNamespace,
	}
}

// GetClientFromSecret returns YandexClient build from secret and key
func (b *YandexClientBuilder) GetClientFromSecret(ctx context.Context, cl kube.Client, secretName string, keyName string) (Client, error) {
	secret := &corev1.Secret{}
	if err := cl.Get(ctx, kube.ObjectKey{Name: secretName, Namespace: b.controllerNamespace}, secret); err != nil {
		return nil, err
	}

	key, ok := secret.Data[keyName]
	if !ok {
		return nil, fmt.Errorf("key %s not found in secret %s/%s", keyName, secretName, metav1.NamespaceDefault)
	}

	return GetClient(ctx, string(key))
}

// GetDefaultClient returns YandexClient build from default key
func (b *YandexClientBuilder) GetDefaultClient(ctx context.Context) (Client, error) {
	return GetClient(context.Background(), b.defaultKey)
}

// GetClientFromKey returns YandexClient build from provided key
func (b *YandexClientBuilder) GetClientFromKey(ctx context.Context, key string) (Client, error) {
	return GetClient(ctx, key)
}
