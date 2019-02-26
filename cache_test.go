package k8s_acme_cache

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8s_testing "k8s.io/client-go/testing"
)

func TestKubernetesCacheNoSecret(t *testing.T) {

	cli := fake.NewSimpleClientset()

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("Not Authorized")
	})

	cache := KubernetesCache(
		"mysecret",
		"default",
		cli,
		1,
	)

	_, err := cache.Get(context.Background(), "null")
	if err != autocert.ErrCacheMiss {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCacheContextTimeout(t *testing.T) {
	namespace := "default"
	secretName := "myhostcom.secret"

	cli := fake.NewSimpleClientset()
	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, nil
	})

	cache := KubernetesCache(
		secretName,
		namespace,
		cli,
		1,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Nanosecond)
	go func() {
		time.Sleep(1 * time.Microsecond)
		cancel()
	}()

	_, err := cache.Get(ctx, "null")
	if err != ctx.Err() {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCacheGetSuccess(t *testing.T) {
	namespace := "default"
	secretName := "myhostcom.secret"
	secretData := []byte("VHVlIEFwciAyNSAxMzoxMTozNSBFRFQgMjAxNw==")
	dataName := "myhost.com"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dataName: secretData,
		},
		Type: v1.SecretType("Opaque"),
	}

	cli := fake.NewSimpleClientset(secret)

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, secret, nil
	})

	cache := KubernetesCache(
		secretName,
		namespace,
		cli,
		1,
	)
	data, err := cache.Get(context.Background(), dataName)

	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}

	if !bytes.Equal(data, secretData) {
		t.Errorf("Secret material is different!\n    got(%s)\n    want(%s)\n",
			data,
			secretData,
		)
	}
}

func TestKubernetesCachePutNoSecret(t *testing.T) {

	cli := fake.NewSimpleClientset()

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("Doesn't Exist")
	})

	cache := KubernetesCache(
		"mysecret",
		"default",
		cli,
		1,
	)

	err := cache.Put(context.Background(), "data", []byte("data"))
	if err.Error() != `secrets "mysecret" not found` {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCachePutTimeout(t *testing.T) {
	namespace := "default"
	secretName := "myhostcom.secret"
	secretData := []byte("VHVlIEFwciAyNSAxMzoxMTozNSBFRFQgMjAxNw==")
	dataName := "myhost.com"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dataName: secretData,
		},
		Type: v1.SecretType("Opaque"),
	}

	cli := fake.NewSimpleClientset(secret)

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		time.Sleep(time.Millisecond * 50)
		return true, &v1.Secret{}, nil
	})
	cli.AddReactor("put", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, secret, nil
	})

	cache := KubernetesCache(
		secretName,
		namespace,
		cli,
		1,
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond*10)
	go cancel()
	err := cache.Put(ctx, dataName, secretData)

	if err != ctx.Err() {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCachePutSuccess(t *testing.T) {
	namespace := "default"
	secretName := "myhostcom.secret"
	secretData := []byte("VHVlIEFwciAyNSAxMzoxMTozNSBFRFQgMjAxNw==")
	dataName := "myhost.com"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dataName: secretData,
		},
		Type: v1.SecretType("Opaque"),
	}

	cli := fake.NewSimpleClientset(secret)

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, nil
	})

	cli.AddReactor("put", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, secret, nil
	})

	cache := KubernetesCache(
		secretName,
		namespace,
		cli,
		1,
	)
	err := cache.Put(context.Background(), dataName, secretData)

	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCacheDeleteNoSecret(t *testing.T) {

	cli := fake.NewSimpleClientset()

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("Doesn't Exist")
	})

	cache := KubernetesCache(
		"mysecret",
		"default",
		cli,
		1,
	)

	err := cache.Delete(context.Background(), "data")
	if err.Error() != `secrets "mysecret" not found` {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCacheDeleteTimeout(t *testing.T) {
	namespace := "default"
	secretName := "myhostcom.secret"
	secretData := []byte("VHVlIEFwciAyNSAxMzoxMTozNSBFRFQgMjAxNw==")
	dataName := "myhost.com"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dataName: secretData,
		},
		Type: v1.SecretType("Opaque"),
	}

	cli := fake.NewSimpleClientset(secret)

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		time.Sleep(time.Millisecond * 50)
		return true, &v1.Secret{}, nil
	})
	cli.AddReactor("delete", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, secret, nil
	})

	cache := KubernetesCache(
		secretName,
		namespace,
		cli,
		1,
	)
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond*10)
	go cancel()
	err := cache.Delete(ctx, dataName)

	if err != ctx.Err() {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestKubernetesCacheDeleteSuccess(t *testing.T) {
	namespace := "default"
	secretName := "myhostcom.secret"
	secretData := []byte("VHVlIEFwciAyNSAxMzoxMTozNSBFRFQgMjAxNw==")
	dataName := "myhost.com"

	secret := &v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			dataName: secretData,
		},
		Type: v1.SecretType("Opaque"),
	}

	cli := fake.NewSimpleClientset(secret)

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, &v1.Secret{}, nil
	})

	cli.AddReactor("delete", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, secret, nil
	})

	cache := KubernetesCache(
		secretName,
		namespace,
		cli,
		0,
	)
	err := cache.Delete(context.Background(), dataName)

	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}
