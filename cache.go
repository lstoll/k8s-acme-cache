package k8sacmecache

import (
	"context"
	"encoding/base64"

	"golang.org/x/crypto/acme/autocert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type kubernetesCache struct {
	Namespace         string
	SecretName        string
	Client            kubernetes.Interface
	deleteGracePeriod int64
}

// KubernetesCache returns an autocert.Cache that will store the certificate as
// a secret in Kubernetes. It accepts a secret name, namespace,
// kubrenetes.Clientset, and grace period (in seconds)
func New(namespace, secretName string, client kubernetes.Interface, deleteGracePeriod int64) autocert.Cache {
	return &kubernetesCache{
		Namespace:         namespace,
		SecretName:        secretName,
		Client:            client,
		deleteGracePeriod: deleteGracePeriod,
	}
}

func (k *kubernetesCache) Get(ctx context.Context, name string) ([]byte, error) {
	var (
		data []byte
		done = make(chan struct{})
		err  error
	)

	go func() {
		var secret *v1.Secret
		secret, err = k.Client.CoreV1().Secrets(k.Namespace).Get(k.SecretName, metav1.GetOptions{})
		defer close(done)
		if err != nil {
			return
		}
		var ok bool
		data, ok = secret.Data[secretKey(name)]
		if !ok {
			err = autocert.ErrCacheMiss
		}
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

func (k *kubernetesCache) Put(ctx context.Context, name string, data []byte) error {
	var (
		err  error
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		var secret *v1.Secret

		secret, err = k.Client.CoreV1().Secrets(k.Namespace).Get(k.SecretName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				secret, err = k.Client.CoreV1().Secrets(k.Namespace).Create(&v1.Secret{
					ObjectMeta: metav1.ObjectMeta{Namespace: k.Namespace, Name: k.SecretName},
				})
				if err != nil {
					return
				}
			} else {
				return
			}
		}
		if secret.Data == nil {
			secret.Data = map[string][]byte{}
		}
		secret.Data[secretKey(name)] = data

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

func (k *kubernetesCache) Delete(ctx context.Context, name string) error {
	var (
		err  error
		done = make(chan struct{})
	)
	go func() {
		defer close(done)
		var secret *v1.Secret

		secret, err = k.Client.CoreV1().Secrets(k.Namespace).Get(k.SecretName, metav1.GetOptions{})
		if err != nil {
			return
		}
		delete(secret.Data, secretKey(name))

		if len(secret.Data) > 0 {
			return // other cached keys
		}

		select {
		case <-ctx.Done():
		default:
			var (
				orphanDependents = false
			)
			// Don't overwrite the secret if the context was canceled.
			err = k.Client.CoreV1().Secrets(k.Namespace).Delete(k.SecretName, &metav1.DeleteOptions{
				GracePeriodSeconds: &k.deleteGracePeriod,
				OrphanDependents:   &orphanDependents,
			})
		}
	}()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-done:
	}
	return err
}

// secretKey returns a kubernetes secret key safe representation of the given
// key.
func secretKey(key string) string {
	return base64.StdEncoding.EncodeToString([]byte(key))
}
