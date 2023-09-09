/*
Copyright 2023 NVIDIA

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var macvlannetworklog = logf.Log.WithName("macvlannetwork-resource")

func (r *MacvlanNetwork) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

//nolint:lll
//+kubebuilder:webhook:path=/validate-mellanox-com-v1alpha1-macvlannetwork,mutating=false,failurePolicy=fail,sideEffects=None,groups=mellanox.com,resources=macvlannetworks,verbs=create;update,versions=v1alpha1,name=vmacvlannetwork.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &MacvlanNetwork{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *MacvlanNetwork) ValidateCreate() error {
	macvlannetworklog.Info("validate create", "name", r.Name)

	// Validation for create call is not required
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *MacvlanNetwork) ValidateUpdate(_ runtime.Object) error {
	macvlannetworklog.Info("validate update", "name", r.Name)

	// Validation for update call is not required
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *MacvlanNetwork) ValidateDelete() error {
	macvlannetworklog.Info("validate delete", "name", r.Name)

	// Validation for delete call is not required
	return nil
}
