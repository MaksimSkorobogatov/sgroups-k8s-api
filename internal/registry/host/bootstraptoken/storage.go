// Package bootstraptoken implements the hosts/bootstrap-token subresource: a POST proxies
// SGroupsAuthnAPI.IssueBootstrapToken for the host in the URL and returns the short-lived JWT.
package bootstraptoken

import (
	"context"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	sgroupsv1 "github.com/PRO-Robotech/sgroups-proto/pkg/api/sgroups/v1"

	registryerrors "sgroups.io/sgroups-k8s-api/internal/registry/errors"
	registryoptions "sgroups.io/sgroups-k8s-api/internal/registry/options"
	"sgroups.io/sgroups-k8s-api/pkg/apis/sgroups/v1alpha1"
	"sgroups.io/sgroups-k8s-api/pkg/client"
)

// Storage serves hosts/bootstrap-token: POST mints a bootstrap token for the host.
type Storage struct {
	client *client.Client
}

func NewStorage(c *client.Client, _ registryoptions.StorageOptions) *Storage {
	return &Storage{client: c}
}

var (
	_ rest.Storage   = (*Storage)(nil)
	_ rest.Scoper    = (*Storage)(nil)
	_ rest.Connecter = (*Storage)(nil)
)

func (s *Storage) NamespaceScoped() bool    { return true }
func (s *Storage) New() runtime.Object      { return &v1alpha1.BootstrapToken{} }
func (s *Storage) Destroy()                 {}
func (s *Storage) ConnectMethods() []string { return []string{http.MethodPost} }

// NewConnectOptions returns nil — the request carries no options; the host is taken from the URL.
func (s *Storage) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, false, ""
}

// Connect mints a bootstrap token for the host identified by the URL: namespace from the request
// context, name (id) from the path.
func (s *Storage) Connect(ctx context.Context, id string, _ runtime.Object, responder rest.Responder) (http.Handler, error) {
	namespace, _ := request.NamespaceFrom(ctx)

	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		resp, err := s.client.Authn.IssueBootstrapToken(req.Context(), &sgroupsv1.AuthnReq_BootstrapToken{
			Metadata: &sgroupsv1.AuthnReq_Metadata{Namespace: namespace, Name: id},
		})
		if err != nil {
			// Translate the backend gRPC status into a proper Kubernetes API error so the
			// aggregation layer returns the right HTTP code (e.g. NotFound -> 404) with a
			// clean message, instead of a generic 500.
			gr := v1alpha1.SchemeGroupVersion.WithResource("hosts").GroupResource()
			responder.Error(registryerrors.FromGRPC(err, gr, id))

			return
		}
		responder.Object(http.StatusOK, &v1alpha1.BootstrapToken{
			TypeMeta: metav1.TypeMeta{Kind: v1alpha1.KindBootstrapToken, APIVersion: v1alpha1.SchemeGroupVersion.String()},
			Token:    resp.GetToken(),
		})
	}), nil
}
