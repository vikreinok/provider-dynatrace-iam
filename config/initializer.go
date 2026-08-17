package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DynatraceImportInitializer checks the target system (Dynatrace) to see if a resource
// already exists. If found, it automatically sets the external-name annotation (UUID)
// to prevent "400 Bad Request" errors on duplicate creation.
type DynatraceImportInitializer struct {
	kube         client.Client
	resourceName string
}

// NewDynatraceImportInitializer creates a new initializer for a specific resource type
func NewDynatraceImportInitializer(c client.Client, resourceName string) managed.Initializer {
	return &DynatraceImportInitializer{
		kube:         c,
		resourceName: resourceName,
	}
}

func (i *DynatraceImportInitializer) Initialize(ctx context.Context, mg xpresource.Managed) error {
	if meta.GetExternalName(mg) != "" {
		return nil
	}

	u, err := toUnstructured(mg)
	if err != nil {
		return err
	}

	httpClient, accountID, err := i.buildHTTPClient(ctx, u)
	if err != nil {
		return err
	}

	return i.lookupAndSetExternalName(ctx, mg, u, httpClient, accountID)
}

func toUnstructured(mg xpresource.Managed) (*unstructured.Unstructured, error) {
	raw, err := json.Marshal(mg)
	if err != nil {
		return nil, errors.Wrap(err, "cannot marshal managed resource to JSON")
	}
	u := &unstructured.Unstructured{}
	if err := u.UnmarshalJSON(raw); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal JSON to unstructured")
	}
	return u, nil
}

func (i *DynatraceImportInitializer) buildHTTPClient(ctx context.Context, u *unstructured.Unstructured) (*http.Client, string, error) {
	creds, err := i.getCredentials(ctx, u)
	if err != nil {
		return nil, "", errors.Wrap(err, "cannot retrieve Dynatrace credentials for initializer")
	}

	accountID := firstNonEmpty(creds["iam_account_id"], creds["dt_account_id"])
	clientID := firstNonEmpty(creds["iam_client_id"], creds["dt_client_id"])
	clientSecret := firstNonEmpty(creds["iam_client_secret"], creds["dt_client_secret"])

	if accountID == "" || clientID == "" || clientSecret == "" {
		return nil, "", errors.New("missing account_id, client_id, or client_secret in credentials secret")
	}

	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     "https://sso.dynatrace.com/sso/oauth2/token",
	}
	return conf.Client(ctx), accountID, nil
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func (i *DynatraceImportInitializer) lookupAndSetExternalName(ctx context.Context, mg xpresource.Managed, u *unstructured.Unstructured, httpClient *http.Client, accountID string) error {
	name := firstNonEmpty(
		getSpecString(u, "forProvider", "name"),
		getSpecString(u, "initProvider", "name"),
	)
	if name == "" {
		return nil
	}

	var uuid string
	var err error

	switch i.resourceName {
	case "dynatrace_iam_group":
		uuid, err = lookupGroup(ctx, httpClient, accountID, name)
	case "dynatrace_iam_policy_boundary":
		uuid, err = lookupPolicyBoundary(ctx, httpClient, accountID, name)
	}

	if err != nil {
		return err
	}
	if uuid != "" {
		meta.SetExternalName(mg, uuid)
	}
	return nil
}

// getCredentials dynamically retrieves credentials secret reference from ProviderConfig
func (i *DynatraceImportInitializer) getCredentials(ctx context.Context, u *unstructured.Unstructured) (map[string]string, error) {
	pcName := firstNonEmpty(
		getSpecString(u, "providerConfigRef", "name"),
		getSpecString(u, "initProvider", "providerConfigRef", "name"),
	)
	if pcName == "" {
		return nil, errors.New("provider config reference name is nil")
	}

	group := "dynatrace.crossplane.io"
	if strings.Contains(u.GroupVersionKind().Group, ".m.dynatrace") ||
		strings.Contains(u.GroupVersionKind().Group, "dynatrace.m") {
		group = "dynatrace.m.crossplane.io"
	}

	pc := &unstructured.Unstructured{}
	pc.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   group,
		Version: "v1beta1",
		Kind:    "ProviderConfig",
	})

	if err := i.kube.Get(ctx, client.ObjectKey{Name: pcName}, pc); err != nil {
		return nil, errors.Wrap(err, "cannot get provider config")
	}

	return i.extractSecretCredentials(ctx, pc)
}

func (i *DynatraceImportInitializer) extractSecretCredentials(ctx context.Context, pc *unstructured.Unstructured) (map[string]string, error) {
	source, _, _ := unstructured.NestedString(pc.Object, "spec", "credentials", "source")
	if source != "Secret" {
		return nil, fmt.Errorf("unsupported credentials source: %s", source)
	}

	secName, _, _ := unstructured.NestedString(pc.Object, "spec", "credentials", "secretRef", "name")
	secNamespace, _, _ := unstructured.NestedString(pc.Object, "spec", "credentials", "secretRef", "namespace")
	secKey, _, _ := unstructured.NestedString(pc.Object, "spec", "credentials", "secretRef", "key")
	if secNamespace == "" {
		secNamespace = "crossplane-system"
	}
	if secName == "" || secKey == "" {
		return nil, errors.New("provider config credentials secretRef name or key is missing")
	}

	secret := &corev1.Secret{}
	if err := i.kube.Get(ctx, client.ObjectKey{Name: secName, Namespace: secNamespace}, secret); err != nil {
		return nil, errors.Wrap(err, "cannot get credentials secret")
	}

	dataBytes, ok := secret.Data[secKey]
	if !ok {
		return nil, fmt.Errorf("secret key %s not found in secret %s/%s", secKey, secNamespace, secName)
	}

	creds := map[string]string{}
	if err := json.Unmarshal(dataBytes, &creds); err != nil {
		return nil, errors.Wrap(err, "cannot parse credentials JSON")
	}

	return creds, nil
}

func getSpecString(u *unstructured.Unstructured, fields ...string) string {
	val, ok, err := unstructured.NestedString(u.Object, append([]string{"spec"}, fields...)...)
	if err != nil || !ok {
		return ""
	}
	return val
}

func lookupGroup(ctx context.Context, client *http.Client, accountID, name string) (string, error) {
	url := fmt.Sprintf("https://api.dynatrace.com/iam/v1/accounts/%s/groups", accountID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code %d querying groups: %s", resp.StatusCode, string(b))
	}

	var response struct {
		Items []struct {
			UUID string `json:"uuid"`
			Name string `json:"name"`
		} `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", err
	}

	for _, g := range response.Items {
		if g.Name == name {
			return g.UUID, nil
		}
	}
	return "", nil
}

func lookupPolicyBoundary(ctx context.Context, client *http.Client, accountID, name string) (string, error) {
	page := 1
	for {
		foundUUID, hasMore, err := fetchBoundaryPage(ctx, client, accountID, name, page)
		if err != nil {
			return "", err
		}
		if foundUUID != "" {
			return foundUUID, nil
		}
		if !hasMore {
			break
		}
		page++
	}
	return "", nil
}

func fetchBoundaryPage(ctx context.Context, client *http.Client, accountID, name string, page int) (string, bool, error) {
	url := fmt.Sprintf("https://api.dynatrace.com/iam/v1/repo/account/%s/boundaries?page=%d&page-size=100", accountID, page)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", false, fmt.Errorf("unexpected status code %d querying policy boundaries: %s", resp.StatusCode, string(b))
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", false, err
	}

	items := extractBoundaryItems(response)
	if len(items) == 0 {
		return "", false, nil
	}

	for _, item := range items {
		if itemMap, ok := item.(map[string]any); ok {
			bName, _ := itemMap["name"].(string)
			bUUID, _ := itemMap["uuid"].(string)
			if bName == name && bUUID != "" {
				return bUUID, true, nil
			}
		}
	}
	return "", true, nil
}

func extractBoundaryItems(response map[string]any) []any {
	for _, key := range []string{"policyBoundaryOverviewList", "boundaries", "items", "content"} {
		if rawItems, ok := response[key].([]any); ok {
			return rawItems
		}
	}
	return nil
}
