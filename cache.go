package k8s_acme_cache

import (
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/context"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/pkg/api/v1"
)

type kubernetesCache struct {
	Namespace  string
	SecretName string
	Client     *kubernetes.Clientset
}

// Returns a autocert cache that will store the Certificate as a secret
// in Kubernetes.
func KubernetesCache(secret, namespace string, client *kubernetes.Clientset) autocert.Cache {
	return kubernetesCache{
		Namespace:  namespace,
		SecretName: secret,
		Client:     client,
	}
}

func (k kubernetesCache) Get(ctx context.Context, name string) ([]byte, error) {
	var (
		data []byte
		done = make(chan struct{})
		err  error
	)

	go func() {
		var secret *v1.Secret
		secret, err = k.Client.CoreV1().Secrets(k.Namespace).Get(k.SecretName)
		defer close(done)
		if err != nil {
			return
		}
		data = secret.Data[name]
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
	}
	if err != nil {
		return nil, autocert.ErrCacheMiss
	}
	return data, err
}

func (k kubernetesCache) Put(ctx context.Context, name string, data []byte) error {
	var (
		err  error
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		var secret *v1.Secret

		secret, err = k.Client.CoreV1().Secrets(k.Namespace).Get(k.SecretName)
		if err != nil {
			return
		}
		secret.Data[name] = data

		select {
		case <-ctx.Done():
		default:
			// Don't overwrite the secret if the context was canceled.
			_, err = k.Client.CoreV1().Secrets(k.Namespace).Update(secret)
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return err
}

func (k kubernetesCache) Delete(ctx context.Context, name string) error {
	var (
		err  error
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		var secret *v1.Secret

		secret, err = k.Client.CoreV1().Secrets(k.Namespace).Get(k.SecretName)
		if err != nil {
			return
		}
		delete(secret.Data, name)

		select {
		case <-ctx.Done():
		default:
			// Don't overwrite the secret if the context was canceled.
			_, err = k.Client.CoreV1().Secrets(k.Namespace).Update(secret)
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return err
}
