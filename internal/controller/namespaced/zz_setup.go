// SPDX-FileCopyrightText: 2024 The Crossplane Authors <https://crossplane.io>
//
// SPDX-License-Identifier: Apache-2.0

package controller

import (
	ctrl "sigs.k8s.io/controller-runtime"

	"github.com/crossplane/upjet/v2/pkg/controller"

	group "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/group"
	permission "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/permission"
	policy "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/policy"
	policybindings "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/policybindings"
	policybindingsv2 "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/policybindingsv2"
	policyboundary "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/policyboundary"
	serviceuser "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/serviceuser"
	user "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/iam/user"
	permissionmgmz "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/mgmz/permission"
	bindings "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/policy/bindings"
	policypolicy "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/policy/policy"
	providerconfig "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/providerconfig"
	groupuser "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/user/group"
	useruser "github.com/vikreinok/provider-dynatrace-iam/internal/controller/namespaced/user/user"
)

// Setup creates all controllers with the supplied logger and adds them to
// the supplied manager.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		group.Setup,
		permission.Setup,
		policy.Setup,
		policybindings.Setup,
		policybindingsv2.Setup,
		policyboundary.Setup,
		serviceuser.Setup,
		user.Setup,
		permissionmgmz.Setup,
		bindings.Setup,
		policypolicy.Setup,
		providerconfig.Setup,
		groupuser.Setup,
		useruser.Setup,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupGated creates all controllers with the supplied logger and adds them to
// the supplied manager gated.
func SetupGated(mgr ctrl.Manager, o controller.Options) error {
	for _, setup := range []func(ctrl.Manager, controller.Options) error{
		group.SetupGated,
		permission.SetupGated,
		policy.SetupGated,
		policybindings.SetupGated,
		policybindingsv2.SetupGated,
		policyboundary.SetupGated,
		serviceuser.SetupGated,
		user.SetupGated,
		permissionmgmz.SetupGated,
		bindings.SetupGated,
		policypolicy.SetupGated,
		providerconfig.SetupGated,
		groupuser.SetupGated,
		useruser.SetupGated,
	} {
		if err := setup(mgr, o); err != nil {
			return err
		}
	}
	return nil
}

// SetupWebhookWithManager registers conversion webhooks for all resource kinds in the group.
func SetupWebhookWithManager(mgr ctrl.Manager) error {
	for _, setup := range []func(ctrl.Manager) error{
		group.SetupWebhookWithManager,
		permission.SetupWebhookWithManager,
		policy.SetupWebhookWithManager,
		policybindings.SetupWebhookWithManager,
		policybindingsv2.SetupWebhookWithManager,
		policyboundary.SetupWebhookWithManager,
		serviceuser.SetupWebhookWithManager,
		user.SetupWebhookWithManager,
		permissionmgmz.SetupWebhookWithManager,
		bindings.SetupWebhookWithManager,
		policypolicy.SetupWebhookWithManager,
		providerconfig.SetupWebhookWithManager,
		groupuser.SetupWebhookWithManager,
		useruser.SetupWebhookWithManager,
	} {
		if err := setup(mgr); err != nil {
			return err
		}
	}
	return nil
}
