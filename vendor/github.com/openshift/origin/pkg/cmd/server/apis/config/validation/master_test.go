package validation

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/validation/field"
	kapihelper "k8s.io/kubernetes/pkg/apis/core/helper"

	"github.com/openshift/origin/pkg/cmd/server/apis/config"
	configapi "github.com/openshift/origin/pkg/cmd/server/apis/config"
)

func TestFailingAPIServerArgs(t *testing.T) {
	args := configapi.ExtendedArguments{}
	args["port"] = []string{"invalid-value"}
	args["missing-key"] = []string{"value"}

	// [port: invalid value '[invalid-value]': could not be set: strconv.ParseUint: parsing "invalid-value": invalid syntax flag: invalid value 'missing-key': is not a valid flag]

	validationResults := ValidateAPIServerExtendedArguments(args, nil)
	errs := validationResults.Errors

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, not %v", errs)
	}

	var (
		portErr    *field.Error
		missingErr *field.Error
	)
	for _, err := range errs {
		switch err.Field {
		case "port":
			portErr = err
		case "flag":
			missingErr = err
		}
	}

	if portErr == nil {
		t.Fatalf("missing port")
	}
	if missingErr == nil {
		t.Fatalf("missing missing-key")
	}

	if e, a := "port", portErr.Field; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := "invalid-value", portErr.BadValue.(string); e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := `could not be set: strconv.ParseInt: parsing "invalid-value": invalid syntax`, portErr.Detail; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}

	if e, a := "flag", missingErr.Field; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := "missing-key", missingErr.BadValue.(string); e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := `is not a valid flag`, missingErr.Detail; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
}

func TestFailingControllerArgs(t *testing.T) {
	args := configapi.ExtendedArguments{}
	args["port"] = []string{"invalid-value"}
	args["missing-key"] = []string{"value"}

	// [port: invalid value '[invalid-value]': could not be set: strconv.ParseUint: parsing "invalid-value": invalid syntax flag: invalid value 'missing-key': is not a valid flag]

	errs := ValidateControllerExtendedArguments(args, nil)

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, not %v", errs)
	}

	var (
		portErr    *field.Error
		missingErr *field.Error
	)
	for _, err := range errs {
		switch err.Field {
		case "port":
			portErr = err
		case "flag":
			missingErr = err
		}
	}

	if portErr == nil {
		t.Fatalf("missing port")
	}
	if missingErr == nil {
		t.Fatalf("missing missing-key")
	}

	if e, a := "port", portErr.Field; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := "invalid-value", portErr.BadValue.(string); e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := `could not be set: strconv.ParseInt: parsing "invalid-value": invalid syntax`, portErr.Detail; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}

	if e, a := "flag", missingErr.Field; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := "missing-key", missingErr.BadValue.(string); e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
	if e, a := `is not a valid flag`, missingErr.Detail; e != a {
		t.Errorf("expected %v, got %v", e, a)
	}
}

func TestValidate_ValidateEtcdStorageConfig(t *testing.T) {
	osField := "openShiftStorageVersion"
	kubeField := "kubernetesStorageVersion"
	tests := []struct {
		label                   string
		kubeStorageVersion      string
		openshiftStorageVersion string
		name                    string
		expected                field.ErrorList
	}{
		{
			label:                   "valid levels",
			kubeStorageVersion:      "v1",
			openshiftStorageVersion: "v1",
			expected:                field.ErrorList{},
		},
		{
			label:                   "unknown openshift level",
			kubeStorageVersion:      "v1",
			openshiftStorageVersion: "bogus",
			expected: field.ErrorList{
				field.NotSupported(field.NewPath(osField), "bogus", []string{"v1"}),
			},
		},
		{
			label:                   "unsupported openshift level",
			kubeStorageVersion:      "v1",
			openshiftStorageVersion: "v1beta3",
			expected: field.ErrorList{
				field.NotSupported(field.NewPath(osField), "v1beta3", []string{"v1"}),
			},
		},
		{
			label:                   "missing openshift level",
			kubeStorageVersion:      "v1",
			openshiftStorageVersion: "",
			expected: field.ErrorList{
				field.Required(field.NewPath(osField), ""),
			},
		},
		{
			label:                   "unknown kube level",
			kubeStorageVersion:      "bogus",
			openshiftStorageVersion: "v1",
			expected: field.ErrorList{
				field.NotSupported(field.NewPath(kubeField), "bogus", []string{"v1"}),
			},
		},
		{
			label:                   "unsupported kube level",
			kubeStorageVersion:      "v1beta3",
			openshiftStorageVersion: "v1",
			expected: field.ErrorList{
				field.NotSupported(field.NewPath(kubeField), "v1beta3", []string{"v1"}),
			},
		},
		{
			label:                   "missing kube level",
			kubeStorageVersion:      "",
			openshiftStorageVersion: "v1",
			expected: field.ErrorList{
				field.Required(field.NewPath(kubeField), ""),
			},
		},
	}

	for _, test := range tests {
		t.Logf("evaluating test: %s", test.label)
		config := config.EtcdStorageConfig{
			OpenShiftStorageVersion:  test.openshiftStorageVersion,
			KubernetesStorageVersion: test.kubeStorageVersion,
		}
		results := ValidateEtcdStorageConfig(config, nil)
		if !kapihelper.Semantic.DeepEqual(test.expected, results) {
			t.Errorf("unexpected validation results; diff:\n%v", diff.ObjectDiff(test.expected, results))
			return
		}
	}
}

func TestValidateAdmissionPluginConfig(t *testing.T) {
	locationOnly := configapi.AdmissionPluginConfig{
		Location: "/some/location",
	}
	configOnly := configapi.AdmissionPluginConfig{
		Configuration: &configapi.NodeConfig{},
	}
	locationAndConfig := configapi.AdmissionPluginConfig{
		Location:      "/some/location",
		Configuration: &configapi.NodeConfig{},
	}
	bothEmpty := configapi.AdmissionPluginConfig{}

	tests := []struct {
		config        map[string]*configapi.AdmissionPluginConfig
		expectError   bool
		warningFields []string
	}{
		{
			config: map[string]*configapi.AdmissionPluginConfig{
				"one": &locationOnly,
				"two": &configOnly,
			},
		},
		{
			config: map[string]*configapi.AdmissionPluginConfig{
				"one": &locationOnly,
				"two": &locationAndConfig,
			},
			expectError: true,
		},
		{
			config: map[string]*configapi.AdmissionPluginConfig{
				"one": &configOnly,
				"two": &bothEmpty,
			},
			expectError: true,
		},
		{
			config: map[string]*configapi.AdmissionPluginConfig{
				"openshift.io/OriginResourceQuota": &configOnly,
				"two":                              &configOnly,
			},
			warningFields: []string{"[openshift.io/OriginResourceQuota]"},
			expectError:   false,
		},
	}

	for _, tc := range tests {
		results := ValidateAdmissionPluginConfig(tc.config, nil)
		if len(results.Errors) > 0 && !tc.expectError {
			t.Errorf("Unexpected error for %#v: %v", tc.config, results.Errors)
		}
		if len(results.Errors) == 0 && tc.expectError {
			t.Errorf("Did not get expected error for: %#v", tc.config)
		}
		actualWarnings := sets.NewString()
		expectedWarnings := sets.NewString(tc.warningFields...)
		for i := range results.Warnings {
			actualWarnings.Insert(results.Warnings[i].Field)
		}
		if !expectedWarnings.Equal(actualWarnings) {
			t.Errorf("Expected warnings: %v, actual warnings: %v", expectedWarnings, actualWarnings)
		}
	}
}

func TestValidateAdmissionPluginConfigConflicts(t *testing.T) {
	testCases := []struct {
		name    string
		options configapi.MasterConfig

		warningFields []string
		errorFields   []string
	}{
		{
			name: "stock everything",
		},
		{
			name: "specified kube admission order 02",
			options: configapi.MasterConfig{
				KubernetesMasterConfig: configapi.KubernetesMasterConfig{
					APIServerArguments: configapi.ExtendedArguments{
						"admission-control": []string{"foo"},
					},
				},
			},
			warningFields: []string{"kubernetesMasterConfig.apiServerArguments[admission-control]"},
		},
		{
			name: "specified origin admission order",
			options: configapi.MasterConfig{
				AdmissionConfig: configapi.AdmissionConfig{
					PluginOrderOverride: []string{"foo"},
				},
			},
			warningFields: []string{"admissionConfig.pluginOrderOverride"},
		},
		{
			name: "specified kube admission config file",
			options: configapi.MasterConfig{
				KubernetesMasterConfig: configapi.KubernetesMasterConfig{
					APIServerArguments: configapi.ExtendedArguments{
						"admission-control-config-file": []string{"foo"},
					},
				},
			},
			warningFields: []string{"kubernetesMasterConfig.apiServerArguments[admission-control-config-file]"},
		},
		{
			name: "specified, non-conflicting plugin configs 01",
			options: configapi.MasterConfig{
				AdmissionConfig: configapi.AdmissionConfig{
					PluginConfig: map[string]*configapi.AdmissionPluginConfig{
						"foo": {
							Location: "bar",
						},
					},
				},
			},
		},
	}

	// these fields have warnings in the empty case
	defaultWarningFields := sets.NewString(
		"serviceAccountConfig.managedNames", "serviceAccountConfig.publicKeyFiles", "serviceAccountConfig.privateKeyFile", "serviceAccountConfig.masterCA",
		"projectConfig.securityAllocator", "kubernetesMasterConfig.proxyClientInfo", "auditConfig.auditFilePath", "aggregatorConfig.proxyClientInfo", "controllerConfig.serviceServingCert.signer")

	for _, tc := range testCases {
		results := ValidateMasterConfig(&tc.options, nil)

		for _, result := range results.Warnings {
			if defaultWarningFields.Has(result.Field) {
				continue
			}

			found := false
			for _, expectedField := range tc.warningFields {
				if result.Field == expectedField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("%s: didn't expect %q", tc.name, result.Field)
			}
		}

		for _, expectedField := range tc.warningFields {
			found := false
			for _, result := range results.Warnings {
				if result.Field == expectedField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("%s: didn't find %q", tc.name, expectedField)
			}
		}

		for _, expectedField := range tc.errorFields {
			found := false
			for _, result := range results.Errors {
				if result.Field == expectedField {
					found = true
					break
				}
			}

			if !found {
				t.Errorf("%s: didn't find %q", tc.name, expectedField)
			}
		}

	}
}

func TestValidateIngressIPNetworkCIDR(t *testing.T) {
	testCases := []struct {
		testName      string
		cidr          string
		serviceCIDR   string
		clusterCIDR   string
		cloudProvider string
		errorCount    int
	}{
		{
			testName: "No CIDR",
		},
		{
			testName:   "Invalid CIDR",
			cidr:       "foo",
			errorCount: 1,
		},
		{
			testName:    "No cloud provider and conflicting CIDRs",
			cidr:        "172.16.0.0/16",
			serviceCIDR: "172.16.0.0/16",
			clusterCIDR: "172.16.0.0/16",
			errorCount:  2,
		},
		{
			testName: "No cloud provider and unspecified CIDR",
			cidr:     "0.0.0.0/32",
		},
		{
			testName:    "No cloud provider and non-conflicting CIDR",
			cidr:        "172.16.0.0/16",
			serviceCIDR: "172.17.0.0/16",
			clusterCIDR: "172.18.0.0/16",
		},
		{
			testName:      "Cloud provider and unspecified CIDR",
			cidr:          "0.0.0.0/32",
			cloudProvider: "foo",
		},
		{
			testName:      "Cloud provider and CIDR",
			cidr:          "172.16.0.0/16",
			cloudProvider: "foo",
			errorCount:    1,
		},
	}
	for _, test := range testCases {
		config := &configapi.MasterConfig{
			KubernetesMasterConfig: configapi.KubernetesMasterConfig{
				ControllerArguments: configapi.ExtendedArguments{
					"cloud-provider": []string{test.cloudProvider},
				},
			},
			NetworkConfig: configapi.MasterNetworkConfig{
				IngressIPNetworkCIDR: test.cidr,
				ServiceNetworkCIDR:   test.serviceCIDR,
				ClusterNetworks: []configapi.ClusterNetworkEntry{
					{
						CIDR: test.clusterCIDR,
					},
				},
			},
		}
		errors := ValidateIngressIPNetworkCIDR(config, nil)
		errorCount := len(errors)
		if test.errorCount != errorCount {
			t.Errorf("%s: expected %d errors, got %d", test.testName, test.errorCount, errorCount)
		}
	}
}

func TestValidateMasterAuthConfig(t *testing.T) {
	testConfigFile, err := ioutil.TempFile("", "test1.cfg")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(testConfigFile.Name())

	metadataFile, err := ioutil.TempFile("", "oauth.metadata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(metadataFile.Name())
	ioutil.WriteFile(metadataFile.Name(), testMetadataContent, os.FileMode(0644))
	badMetadataFile, err := ioutil.TempFile("", "badoauth.metadata")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer os.Remove(badMetadataFile.Name())
	ioutil.WriteFile(badMetadataFile.Name(), []byte("bad file"), os.FileMode(0644))

	testCases := []struct {
		testName                   string
		RequestHeader              *configapi.RequestHeaderAuthenticationOptions
		WebhookTokenAuthenticators []configapi.WebhookTokenAuthenticator
		OAuthMetadataFile          string
		expectedErrors             []string
	}{
		{
			testName: "No AuthConfig",
		},
		{
			testName: "Valid RequestHeader",
			RequestHeader: &configapi.RequestHeaderAuthenticationOptions{
				ClientCA:            "ClientCA",
				ClientCommonNames:   []string{"ClientCommonNames", "2", "3"},
				UsernameHeaders:     []string{"Username", "Headers"},
				GroupHeaders:        []string{"Group", "Headers"},
				ExtraHeaderPrefixes: []string{"Extra", "Header", "Prefixes"},
			},
		},
		{
			testName: "Only One WebhookTokenAuthenticator",
			WebhookTokenAuthenticators: []configapi.WebhookTokenAuthenticator{
				{
					ConfigFile: testConfigFile.Name(),
					CacheTTL:   "2m",
				},
			},
		},
		{
			testName: "Unexisting config file in second WebhookTokenAuthenticator",
			WebhookTokenAuthenticators: []configapi.WebhookTokenAuthenticator{
				{
					ConfigFile: testConfigFile.Name(),
					CacheTTL:   "2m",
				},
				{
					ConfigFile: "Unexisting",
					CacheTTL:   "2m",
				},
			},
			expectedErrors: []string{"webhookTokenAuthenticators.ConfigFile: Invalid value: \"Unexisting\": could not read file: stat Unexisting: no such file or directory"},
		},
		{
			testName: "Invalid and Empty CacheTTL WebhookTokenAuthenticator",
			WebhookTokenAuthenticators: []configapi.WebhookTokenAuthenticator{
				{
					ConfigFile: testConfigFile.Name(),
					CacheTTL:   "-2m",
				},
				{
					ConfigFile: testConfigFile.Name(),
				},
			},
			expectedErrors: []string{
				"webhookTokenAuthenticators.cacheTTL: Invalid value: \"-2m\": cannot be less than zero",
				"webhookTokenAuthenticators.cacheTTL: Required value",
			},
		},
		{
			testName:          "No OAuth Metadata file",
			OAuthMetadataFile: "NoFile",
			expectedErrors:    []string{`oauthMetadataFile: Invalid value: "NoFile": Metadata validation failed: unable to read External OAuth Metadata file: open NoFile: no such file or directory`},
		},
		{
			testName:          "Bad Metadata file",
			OAuthMetadataFile: badMetadataFile.Name(),
			expectedErrors:    []string{fmt.Sprintf(`oauthMetadataFile: Invalid value: %q: Metadata validation failed: unable to decode External OAuth Metadata file: invalid character 'b' looking for beginning of value`, badMetadataFile.Name())},
		},
	}
	for _, test := range testCases {
		config := configapi.MasterAuthConfig{
			RequestHeader:              test.RequestHeader,
			WebhookTokenAuthenticators: test.WebhookTokenAuthenticators,
			OAuthMetadataFile:          test.OAuthMetadataFile,
		}
		errors := ValidateMasterAuthConfig(config, nil)
		if len(test.expectedErrors) != len(errors.Errors) {
			t.Errorf("%s: expected %v errors, got %v", test.testName, test.expectedErrors, errors.Errors)
			continue
		}
		for i := range test.expectedErrors {
			if errors.Errors[i].Error() != test.expectedErrors[i] {
				t.Errorf("%s: expected error '%s', got '%s'", test.testName, test.expectedErrors[i], errors.Errors[i])
			}
		}
	}
}

var testMetadataContent = []byte(`{
	"issuer": "https://127.0.0.1/",
	"authorization_endpoint": "https://127.0.0.1/",
	"token_endpoint": "https://127.0.0.1/",
	"scopes_supported": ["openid", "profile", "email", "address", "phone", "offline_access"],
	"response_types_supported": ["code", "code token"],
	"grant_types_supported": ["authorization_code", "implicit"],
	"code_challenge_methods_supported": ["plain", "S256"]}`)
