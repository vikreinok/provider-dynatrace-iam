// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package costcenter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/crossplane/crossplane-runtime/v2/pkg/event"
	"github.com/crossplane/crossplane-runtime/v2/pkg/meta"
	"github.com/crossplane/crossplane-runtime/v2/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/v2/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/v2/pkg/resource"
	xpv1 "github.com/crossplane/crossplane/apis/v2/core/v2"
	tjcontroller "github.com/crossplane/upjet/v2/pkg/controller"
	"github.com/pkg/errors"
	"golang.org/x/oauth2/clientcredentials"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	v1alpha1 "github.com/vikreinok/provider-dynatrace-iam/apis/cluster/iam/v1alpha1"
	clusterv1beta1 "github.com/vikreinok/provider-dynatrace-iam/apis/cluster/v1beta1"
)

const (
	errNotCostCenter = "managed resource is not a CostCenter custom resource"
	errNewClient     = "cannot create Dynatrace Account API client"

	accountApiUrl = "https://api.dynatrace.com/v1"
	tokenUrl      = "https://sso.dynatrace.com/sso/oauth2/token"
)

// SetupGated adds a controller that reconciles CostCenter managed resources.
func SetupGated(mgr ctrl.Manager, o tjcontroller.Options) error {
	o.Gate.Register(func() {
		if err := Setup(mgr, o); err != nil {
			mgr.GetLogger().Error(err, "unable to setup reconciler", "gvk", v1alpha1.CostCenter_GroupVersionKind.String())
		}
	}, v1alpha1.CostCenter_GroupVersionKind)
	return nil
}

// Setup adds a controller that reconciles CostCenter managed resources.
func Setup(mgr ctrl.Manager, o tjcontroller.Options) error {
	name := managed.ControllerName(v1alpha1.CostCenter_GroupKind)

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.CostCenter_GroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			newServiceFn: newService,
		}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithPollInterval(o.PollInterval),
	)

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		For(&v1alpha1.CostCenter{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube         client.Client
	newServiceFn func(ctx context.Context, clientID, clientSecret, accountID string) (service, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.CostCenter)
	if !ok {
		return nil, errors.New(errNotCostCenter)
	}

	// For cluster-scoped resources, we use ProviderConfig from cluster package
	pc := &clusterv1beta1.ProviderConfig{}
	if err := c.kube.Get(ctx, client.ObjectKey{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, "cannot get provider config")
	}

	// Extract credentials
	data, err := resource.CommonCredentialExtractor(ctx, pc.Spec.Credentials.Source, c.kube, pc.Spec.Credentials.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, "cannot extract credentials")
	}
	creds := map[string]string{}
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, errors.Wrap(err, "cannot unmarshal credentials")
	}

	accountID := creds["iam_account_id"]
	if accountID == "" {
		accountID = creds["dt_account_id"]
	}
	clientID := creds["iam_client_id"]
	if clientID == "" {
		clientID = creds["dt_client_id"]
	}
	clientSecret := creds["iam_client_secret"]
	if clientSecret == "" {
		clientSecret = creds["dt_client_secret"]
	}

	svc, err := c.newServiceFn(ctx, clientID, clientSecret, accountID)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	return &external{kube: c.kube, service: svc}, nil
}

type external struct {
	kube    client.Client
	service service
}

func (e *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.CostCenter)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotCostCenter)
	}

	key := meta.GetExternalName(cr)
	targetKey := key
	if cr.Spec.ForProvider.Key != nil && *cr.Spec.ForProvider.Key != "" {
		targetKey = *cr.Spec.ForProvider.Key
	}

	if key == "" && targetKey == "" {
		return managed.ExternalObservation{ResourceExists: false}, nil
	}

	list, err := e.service.List(ctx)
	if err != nil {
		return managed.ExternalObservation{}, err
	}

	for _, item := range list {
		if (key != "" && item.Key == key) || (targetKey != "" && item.Key == targetKey) {
			if meta.GetExternalName(cr) != item.Key {
				meta.SetExternalName(cr, item.Key)
				if err := e.kube.Update(ctx, cr); err != nil {
					return managed.ExternalObservation{}, errors.Wrap(err, "cannot update external-name annotation")
				}
			}
			cr.Status.AtProvider.Key = &item.Key
			cr.Status.AtProvider.Value = &item.Value
			cr.Status.AtProvider.ID = &item.Key
			cr.SetConditions(xpv1.Available())
			return managed.ExternalObservation{
				ResourceExists:    true,
				ResourceUpToDate:  true,
				ConnectionDetails: nil,
			}, nil
		}
	}

	return managed.ExternalObservation{ResourceExists: false}, nil
}

func (e *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.CostCenter)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotCostCenter)
	}

	if cr.Spec.ForProvider.Key == nil {
		return managed.ExternalCreation{}, errors.New("spec.forProvider.key is required")
	}
	key := *cr.Spec.ForProvider.Key
	val := ""
	if cr.Spec.ForProvider.Value != nil {
		val = *cr.Spec.ForProvider.Value
	}

	err := e.service.Create(ctx, key, val)
	if err != nil {
		if strings.Contains(err.Error(), "already been stored") {
			meta.SetExternalName(cr, key)
			if updateErr := e.kube.Update(ctx, cr); updateErr != nil {
				return managed.ExternalCreation{}, errors.Wrap(updateErr, "cannot update external-name annotation after conflict")
			}
			return managed.ExternalCreation{}, nil
		}
		return managed.ExternalCreation{}, err
	}

	meta.SetExternalName(cr, key)
	if updateErr := e.kube.Update(ctx, cr); updateErr != nil {
		return managed.ExternalCreation{}, errors.Wrap(updateErr, "cannot update external-name annotation")
	}
	return managed.ExternalCreation{}, nil
}

func (e *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.CostCenter)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotCostCenter)
	}

	key := meta.GetExternalName(cr)
	if key == "" && cr.Spec.ForProvider.Key != nil {
		key = *cr.Spec.ForProvider.Key
	}
	val := ""
	if cr.Spec.ForProvider.Value != nil {
		val = *cr.Spec.ForProvider.Value
	}

	err := e.service.Update(ctx, key, val)
	return managed.ExternalUpdate{}, err
}

func (e *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.CostCenter)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotCostCenter)
	}

	return managed.ExternalDelete{}, e.service.Delete(ctx, meta.GetExternalName(cr))
}

func (e *external) Disconnect(ctx context.Context) error {
	return nil
}

type costCenterItem struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type service interface {
	List(ctx context.Context) ([]costCenterItem, error)
	Create(ctx context.Context, key, value string) error
	Update(ctx context.Context, key, value string) error
	Delete(ctx context.Context, key string) error
}

type apiService struct {
	client    *http.Client
	accountID string
}

func newService(ctx context.Context, clientID, clientSecret, accountID string) (service, error) {
	conf := &clientcredentials.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		TokenURL:     tokenUrl,
	}
	return &apiService{
		client:    conf.Client(ctx),
		accountID: accountID,
	}, nil
}

func (s *apiService) List(ctx context.Context) ([]costCenterItem, error) {
	url := fmt.Sprintf("%s/accounts/%s/settings/costcenters", accountApiUrl, s.accountID)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(b))
	}

	var response struct {
		Records []costCenterItem `json:"records"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return response.Records, nil
}

func (s *apiService) Create(ctx context.Context, key, value string) error {
	url := fmt.Sprintf("%s/accounts/%s/settings/costcenters", accountApiUrl, s.accountID)
	payload := struct {
		Values []costCenterItem `json:"values"`
	}{
		Values: []costCenterItem{{Key: key, Value: value}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *apiService) Update(ctx context.Context, key, value string) error {
	url := fmt.Sprintf("%s/accounts/%s/settings/costcenters", accountApiUrl, s.accountID)
	payload := struct {
		Values []costCenterItem `json:"values"`
	}{
		Values: []costCenterItem{{Key: key, Value: value}},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (s *apiService) Delete(ctx context.Context, key string) error {
	url := fmt.Sprintf("%s/accounts/%s/settings/costcenters/%s", accountApiUrl, s.accountID, key)
	req, _ := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusNotFound {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(b))
	}
	return nil
}
