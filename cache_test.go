package k8s_acme_cache

import (
	"bytes"
	"errors"
	"flag"
	"testing"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/net/context"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	k8s_testing "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconfig = flag.String("kubeconfig", "", "Cluster to run tests against, if non-empty. Otherwise, use fake clientset")

func TestKubernetesCacheNoSecret(t *testing.T) {

	cli := fake.NewSimpleClientset()

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, nil, errors.New("Not Authorized")
	})

	cache := New(
		"default",
		"mysecret",
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

	cache := New(
		namespace,
		secretName,
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
			secretKey(dataName): secretData,
		},
		Type: v1.SecretType("Opaque"),
	}

	cli := fake.NewSimpleClientset(secret)

	cli.AddReactor("get", "secrets", func(action k8s_testing.Action) (handled bool, ret runtime.Object, err error) {
		return true, secret, nil
	})

	cache := New(
		namespace,
		secretName,
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

	cache := New(
		"default",
		"mysecret",
		cli,
		1,
	)

	err := cache.Put(context.Background(), "data", []byte("data"))
	if err != nil {
		t.Errorf("Unexpected error when putting new secret: %+v", err)
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
			secretKey(dataName): secretData,
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

	cache := New(
		namespace,
		secretName,
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
			secretKey(dataName): secretData,
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

	cache := New(
		namespace,
		secretName,
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

	cache := New(
		"default",
		"mysecret",
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
			secretKey(dataName): secretData,
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

	cache := New(
		namespace,
		secretName,
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
			secretKey(dataName): secretData,
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

	cache := New(
		namespace,
		secretName,
		cli,
		0,
	)
	err := cache.Delete(context.Background(), dataName)

	if err != nil {
		t.Errorf("Unexpected error: %s", err.Error())
	}
}

func TestE2E(t *testing.T) {
	if *kubeconfig == "" {
		t.Skip("kubeconfig not provided")
	}

	ctx := context.Background()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		t.Fatal(err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		t.Fatal(err)
	}

	const (
		ns = "k8s-acme-cache-test"
		n  = "acmecache"
	)

	cache := New(
		ns,
		n,
		clientset,
		0,
	)

	// blindly clean up first.
	_ = clientset.CoreV1().Secrets(ns).Delete(n, &metav1.DeleteOptions{})
	_, err = clientset.CoreV1().Namespaces().Create(&v1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.Fatalf("Couldn't create test namespace: %+v", err)
	}

	_, err = cache.Get(ctx, "test+1")
	if err != autocert.ErrCacheMiss {
		t.Fatalf("Unexpected error when getting key in non-existent secret: %+v", err)
	}

	if err = cache.Put(ctx, "test+1", []byte("hello")); err != nil {
		t.Fatalf("Unexpected error when putting a secret: %+v", err)
	}

	if err = cache.Put(ctx, "test+2", []byte("hello2")); err != nil {
		t.Fatalf("Unexpected error when putting a secret: %+v", err)
	}

	ret, err := cache.Get(ctx, "test+1")
	if err != nil {
		t.Fatalf("Unexpected error when getting a secret: %+v", err)
	}
	if string(ret) != "hello" {
		t.Fatalf("Want \"hello\" got %q", string(ret))
	}

	if err = cache.Delete(ctx, "test+1"); err != nil {
		t.Fatalf("Unexpected error when deleting a secret: %+v", err)
	}

	ret, err = cache.Get(ctx, "test+2")
	if err != nil {
		t.Fatalf("Unexpected error when getting secret after deleting different key: %+v", err)
	}
	if string(ret) != "hello2" {
		t.Fatalf("Want \"hello2\" got %q", string(ret))
	}

	_, err = cache.Get(ctx, "test+3")
	if err != autocert.ErrCacheMiss {
		t.Fatalf("Unexpected error when getting non-existent key in existing secret: %+v", err)
	}
}
