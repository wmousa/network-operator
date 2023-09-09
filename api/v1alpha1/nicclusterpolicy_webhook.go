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
	"encoding/json"
	"math/big"
	"regexp"
	"strings"

	"github.com/xeipuuv/gojsonschema"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var nicclusterpolicylog = logf.Log.WithName("nicclusterpolicy-resource")

func (w *NicClusterPolicy) SetupWebhookWithManager(mgr ctrl.Manager) error {
	nicclusterpolicylog.Info("Nic cluster policy webhook admission controller")
	return ctrl.NewWebhookManagedBy(mgr).
		For(w).
		Complete()
}

//nolint:lll
//+kubebuilder:webhook:path=/validate-mellanox-com-v1alpha1-nicclusterpolicy,mutating=false,failurePolicy=fail,sideEffects=None,groups=mellanox.com,resources=nicclusterpolicies,verbs=create;update,versions=v1alpha1,name=vnicclusterpolicy.kb.io,admissionReviewVersions=v1

var _ webhook.Validator = &NicClusterPolicy{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (w *NicClusterPolicy) ValidateCreate() error {
	nicclusterpolicylog.Info("validate create", "name", w.Name)
	return w.validateNicClusterPolicy()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (w *NicClusterPolicy) ValidateUpdate(_ runtime.Object) error {
	nicclusterpolicylog.Info("validate update", "name", w.Name)
	return w.validateNicClusterPolicy()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (w *NicClusterPolicy) ValidateDelete() error {
	nicclusterpolicylog.Info("validate delete", "name", w.Name)

	// Validation for delete call is not required
	return nil
}

/*
We validate here
*/
func (w *NicClusterPolicy) validateNicClusterPolicy() error {
	var allErrs field.ErrorList
	// Validate IBKubernetes
	ibKubernetes := w.Spec.IBKubernetes
	if ibKubernetes != nil {
		allErrs = append(allErrs, ibKubernetes.validate(field.NewPath("spec").Child("ibKubernetes"))...)
	}

	// Validate OFEDDriverSpec
	ofedDriver := w.Spec.OFEDDriver
	if ofedDriver != nil {
		allErrs = append(allErrs, ofedDriver.validateVersion(field.NewPath("spec").Child("ofedDriver"))...)
	}
	// Validate RdmaSharedDevicePlugin
	rdmaSharedDevicePlugin := w.Spec.RdmaSharedDevicePlugin
	if rdmaSharedDevicePlugin != nil {
		allErrs = append(allErrs, w.Spec.RdmaSharedDevicePlugin.validateRdmaSharedDevicePlugin(
			field.NewPath("spec").Child("rdmaSharedDevicePlugin"))...)
	}
	// Validate SriovDevicePlugin
	sriovNetworkDevicePlugin := w.Spec.SriovDevicePlugin
	if sriovNetworkDevicePlugin != nil {
		allErrs = append(allErrs, w.Spec.SriovDevicePlugin.validateSriovNetworkDevicePlugin(
			field.NewPath("spec").Child("sriovNetworkDevicePlugin"))...)
	}

	if len(allErrs) == 0 {
		return nil
	}
	return apierrors.NewInvalid(
		schema.GroupKind{Group: "mellanox.com", Kind: "NicClusterPolicy"},
		w.Name, allErrs)
}
func (dp *DevicePluginSpec) validateSriovNetworkDevicePlugin(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	var sriovNetworkDevicePluginConfigJSON map[string]interface{}
	sriovNetworkDevicePluginConfig := *dp.Config

	// Validate if the SRIOV Network Device Plugin Config is a valid json
	if err := json.Unmarshal([]byte(sriovNetworkDevicePluginConfig), &sriovNetworkDevicePluginConfigJSON); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config,
			"Invalid json of SriovNetworkDevicePluginConfig"))
		return allErrs
	}

	// Load the JSON Schema
	sriovNetworkDevicePluginSchemaLoader := gojsonschema.NewReferenceLoader(
		"file://./schema/sriov_network_device_plugin_schema.json")
	acceleratorJSONSchemaLoader := gojsonschema.NewReferenceLoader(
		"file://./schema/accelerator_selector_schema.json")
	netDeviceJSONSchemaLoader := gojsonschema.NewReferenceLoader("file://./schema/net_device_schema.json")
	auxNetDeviceJSONSchemaLoader := gojsonschema.NewReferenceLoader("file://./schema/aux_net_device_schema.json")

	// Load the Sriov Network Device Plugin JSON Loader
	sriovNetworkDevicePluginConfigJSONLoader := gojsonschema.NewStringLoader(sriovNetworkDevicePluginConfig)

	// Perform schema validation
	result, err := gojsonschema.Validate(sriovNetworkDevicePluginSchemaLoader,
		sriovNetworkDevicePluginConfigJSONLoader)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config,
			"Invalid json configuration of SriovNetworkDevicePluginConfig"+err.Error()))
		return allErrs
	} else if !result.Valid() {
		for _, ResultErr := range result.Errors() {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config, ResultErr.Description()))
		}
		return allErrs
	}
	if resourceListInterface := sriovNetworkDevicePluginConfigJSON["resourceList"]; resourceListInterface != nil {
		resourceList, _ := resourceListInterface.([]interface{})
		for _, resourceInterface := range resourceList {
			resource := resourceInterface.(map[string]interface{})
			resourceJSONString, _ := json.Marshal(resource)
			resourceJSONLoader := gojsonschema.NewStringLoader(string(resourceJSONString))
			var selectorResult *gojsonschema.Result
			var selectorErr error
			resourceName := resource["resourceName"].(string)
			if !isValidSriovNetworkDevicePluginResourceName(resourceName) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config,
					"Invalid Resource name, it must consist of alphanumeric characters, '_' or '.', "+
						"and must start and end with an alphanumeric character (e.g. 'MyName',  or 'my.name',  "+
						"or '123_abc', regex used for validation is '([A-Za-z0-9][A-Za-z0-9_.]*)?[A-Za-z0-9]')"))
			}
			deviceType := resource["deviceType"]
			switch deviceType {
			case "accelerator":
				selectorResult, selectorErr = gojsonschema.Validate(acceleratorJSONSchemaLoader, resourceJSONLoader)
			case "auxNetDevice":
				selectorResult, selectorErr = gojsonschema.Validate(auxNetDeviceJSONSchemaLoader, resourceJSONLoader)
			default:
				selectorResult, selectorErr = gojsonschema.Validate(netDeviceJSONSchemaLoader, resourceJSONLoader)
			}
			if selectorErr != nil {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config,
					selectorErr.Error()))
			} else if !selectorResult.Valid() {
				for _, selectorResultErr := range selectorResult.Errors() {
					allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config,
						selectorResultErr.Description()))
				}
			}
		}
	}
	return allErrs
}

func (dp *DevicePluginSpec) validateRdmaSharedDevicePlugin(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList
	var rdmaSharedDevicePluginConfigJSON map[string]interface{}
	rdmaSharedDevicePluginConfig := *dp.Config

	// Validate if the RDMA Shared Device Plugin Config is a valid json
	if err := json.Unmarshal([]byte(rdmaSharedDevicePluginConfig), &rdmaSharedDevicePluginConfigJSON); err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"),
			dp.Config, "Invalid json of RdmaSharedDevicePluginConfig"+err.Error()))
		return allErrs
	}

	// Perform schema validation
	rdmaSharedDevicePluginSchemaLoader := gojsonschema.NewReferenceLoader(
		"file://./schema/rdma_shared_device_plugin_schema.json")
	rdmaSharedDevicePluginConfigJSONLoader := gojsonschema.NewStringLoader(rdmaSharedDevicePluginConfig)
	result, err := gojsonschema.Validate(rdmaSharedDevicePluginSchemaLoader, rdmaSharedDevicePluginConfigJSONLoader)
	if err != nil {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config,
			"Invalid json configuration of rdmaSharedDevicePluginConfig"+err.Error()))
	} else if result.Valid() {
		configListInterface := rdmaSharedDevicePluginConfigJSON["configList"]
		configList, _ := configListInterface.([]interface{})
		for _, configInterface := range configList {
			config := configInterface.(map[string]interface{})
			resourceName := config["resourceName"].(string)
			if !isValidRdmaSharedDevicePluginResourceName(resourceName) {
				allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"),
					dp.Config, "Invalid Resource name, it must consist of alphanumeric characters, "+
						"'-', '_' or '.', and must start and end with an alphanumeric character "+
						"(e.g. 'MyName',  or 'my.name',  or '123-abc', regex used for validation is '([A-Za-z0-9][-A-Za-z0"+
						"-9_.]*)?[A-Za-z0-9]')"))
			}
		}
	} else {
		for _, ResultErr := range result.Errors() {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("Config"), dp.Config, ResultErr.Description()))
		}
	}
	return allErrs
}

// validate is a helper function to perform validation for IBKubernetesSpec.
func (ibk *IBKubernetesSpec) validate(fldPath *field.Path) field.ErrorList {
	var allErrs field.ErrorList

	if !isValidPKeyGUID(ibk.PKeyGUIDPoolRangeStart) || !isValidPKeyGUID(ibk.PKeyGUIDPoolRangeEnd) {
		if !isValidPKeyGUID(ibk.PKeyGUIDPoolRangeStart) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("pKeyGUIDPoolRangeStart"),
				ibk.PKeyGUIDPoolRangeStart, "pKeyGUIDPoolRangeStart must be a valid GUID format:"+
					"xx:xx:xx:xx:xx:xx:xx:xx with Hexa numbers"))
		}
		if !isValidPKeyGUID(ibk.PKeyGUIDPoolRangeEnd) {
			allErrs = append(allErrs, field.Invalid(fldPath.Child("pKeyGUIDPoolRangeEnd"),
				ibk.PKeyGUIDPoolRangeEnd, "pKeyGUIDPoolRangeEnd must be a valid GUID format: "+
					"xx:xx:xx:xx:xx:xx:xx:xx with Hexa numbers"))
		}
		return allErrs
	} else if !isValidPKeyRange(ibk.PKeyGUIDPoolRangeStart, ibk.PKeyGUIDPoolRangeEnd) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("pKeyGUIDPoolRangeEnd"),
			ibk.PKeyGUIDPoolRangeEnd, "pKeyGUIDPoolRangeEnd must not be greater than 02:FF:FF:FF:FF:FF:FF:FF"))
	}
	return allErrs
}

// isValidPKeyGUID checks if a given string is a valid GUID format.
func isValidPKeyGUID(guid string) bool {
	PKeyGUIDPattern := `^([0-9A-Fa-f]{2}:){7}([0-9A-Fa-f]{2})$`
	PKeyGUIDRegex := regexp.MustCompile(PKeyGUIDPattern)
	return PKeyGUIDRegex.MatchString(guid)
}

// isValidPKeyRange checks if range of startGUID and endGUID sis valid
func isValidPKeyRange(startGUID, endGUID string) bool {
	startGUIDWithoutSeparator := strings.ReplaceAll(startGUID, ":", "")
	endGUIDWithoutSeparator := strings.ReplaceAll(endGUID, ":", "")

	startGUIDIntValue := new(big.Int)
	endGUIDIntValue := new(big.Int)
	startGUIDIntValue, _ = startGUIDIntValue.SetString(startGUIDWithoutSeparator, 16)
	endGUIDIntValue, _ = endGUIDIntValue.SetString(endGUIDWithoutSeparator, 16)
	return endGUIDIntValue.Cmp(startGUIDIntValue) > 0
}

func (ofedSpec *OFEDDriverSpec) validateVersion(fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	// Perform version validation logic here
	if !isValidOFEDVersion(ofedSpec.Version) {
		allErrs = append(allErrs, field.Invalid(fldPath.Child("version"), ofedSpec.Version,
			"invalid OFED version"))
	}
	return allErrs
}

// isValidOFEDVersion is a custom function to validate OFED version
func isValidOFEDVersion(version string) bool {
	versionPattern := `^(\d+\.\d+-\d+(\.\d+)*)$`
	versionRegex := regexp.MustCompile(versionPattern)
	return versionRegex.MatchString(version)
}

func isValidSriovNetworkDevicePluginResourceName(resourceName string) bool {
	resourceNamePattern := `^([A-Za-z0-9][A-Za-z0-9_.]*)?[A-Za-z0-9]$`
	resourceNameRegex := regexp.MustCompile(resourceNamePattern)
	return resourceNameRegex.MatchString(resourceName)
}

func isValidRdmaSharedDevicePluginResourceName(resourceName string) bool {
	resourceNamePattern := `^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`
	resourceNameRegex := regexp.MustCompile(resourceNamePattern)
	return resourceNameRegex.MatchString(resourceName)
}
