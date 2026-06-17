package tenant

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"

	appbackend "sgroups.io/sgroups-k8s-api/internal/backend"
	"sgroups.io/sgroups-k8s-api/internal/mock"
	registryoptions "sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/internal/registry/testutil"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
)

func TestTenantStorageCreateList(t *testing.T) {
	mb := mock.New()
	b := appbackend.Backend{Namespaces: mb, AddressGroups: mb, Networks: mb}
	cli, cleanup := testutil.NewBufconnClient(t, b)
	defer cleanup()

	store := NewStorage(cli, registryoptions.StorageOptions{})

	obj := &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "acme"},
		Spec: v1alpha1.TenantSpec{
			DisplayName: "Acme",
		},
	}

	created, err := store.Create(context.Background(), obj, nil, &metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	ns, ok := created.(*v1alpha1.Tenant)
	if !ok {
		t.Fatalf("unexpected type: %T", created)
	}
	if ns.Name != "acme" {
		t.Fatalf("unexpected name: %q", ns.Name)
	}
	if ns.ResourceVersion == "" {
		t.Fatalf("expected resourceVersion to be set")
	}

	listObj, err := store.List(context.Background(), &metainternalversion.ListOptions{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	list, ok := listObj.(*v1alpha1.TenantList)
	if !ok {
		t.Fatalf("unexpected list type: %T", listObj)
	}
	if len(list.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(list.Items))
	}
}

func TestTenantStorageWatch(t *testing.T) {
	mb := mock.New()
	b := appbackend.Backend{Namespaces: mb, AddressGroups: mb, Networks: mb}
	cli, cleanup := testutil.NewBufconnClient(t, b)
	defer cleanup()

	store := NewStorage(cli, registryoptions.StorageOptions{})
	ctx := context.Background()

	// Create initial tenant.
	_, err := store.Create(ctx, &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "ns-1"},
		Spec:       v1alpha1.TenantSpec{DisplayName: "NS 1"},
	}, nil, &metav1.CreateOptions{})
	require.NoError(t, err)

	// List to get ResourceVersion for Watch resumption.
	listObj, err := store.List(ctx, &metainternalversion.ListOptions{})
	require.NoError(t, err)
	list := listObj.(*v1alpha1.TenantList)
	rv := list.ResourceVersion
	require.NotEmpty(t, rv)

	// Start watch with ResourceVersion from List.
	watcher, err := store.Watch(ctx, &metainternalversion.ListOptions{ResourceVersion: rv})
	require.NoError(t, err)
	defer watcher.Stop()

	// Read initial snapshot (ADDED event for existing ns-1).
	var evt watch.Event
	require.Eventually(t, func() bool {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return false
			}
			evt = e

			return true
		default:
			return false
		}
	}, 5*time.Second, 50*time.Millisecond, "expected initial ADDED event")
	require.Equal(t, watch.Added, evt.Type)
	ns := evt.Object.(*v1alpha1.Tenant)
	require.Equal(t, "ns-1", ns.Name)

	// Create a second tenant to trigger a watch event.
	_, err = store.Create(ctx, &v1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: "ns-2"},
		Spec:       v1alpha1.TenantSpec{DisplayName: "NS 2"},
	}, nil, &metav1.CreateOptions{})
	require.NoError(t, err)

	// Read the ADDED event for ns-2.
	require.Eventually(t, func() bool {
		select {
		case e, ok := <-watcher.ResultChan():
			if !ok {
				return false
			}
			evt = e

			return true
		default:
			return false
		}
	}, 5*time.Second, 50*time.Millisecond, "expected ADDED event for ns-2")
	require.Equal(t, watch.Added, evt.Type)
	ns = evt.Object.(*v1alpha1.Tenant)
	require.Equal(t, "ns-2", ns.Name)

	// Stop the watcher and verify channel is closed.
	watcher.Stop()
	require.Eventually(t, func() bool {
		_, ok := <-watcher.ResultChan()

		return !ok
	}, 5*time.Second, 50*time.Millisecond, "expected ResultChan to be closed after Stop")
}
