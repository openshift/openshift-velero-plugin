package buildconfig

import (
	"encoding/json"
	"fmt"

	"github.com/fusor/openshift-velero-plugin/velero-plugins/build"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/clients"
	"github.com/fusor/openshift-velero-plugin/velero-plugins/common"
	"github.com/heptio/velero/pkg/plugin/velero"
	buildv1API "github.com/openshift/api/build/v1"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// RestorePlugin is a restore item action plugin for Velero
type RestorePlugin struct {
	Log logrus.FieldLogger
}

// AppliesTo returns a velero.ResourceSelector that applies to buildconfigs
func (p *RestorePlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"buildconfigs"},
	}, nil
}

// Execute action for the restore plugin for the buildconfig resource
func (p *RestorePlugin) Execute(input *velero.RestoreItemActionExecuteInput) (*velero.RestoreItemActionExecuteOutput, error) {
	p.Log.Info("[buildconfig-restore] Entering buildconfig restore plugin")

	buildconfig := buildv1API.BuildConfig{}
	itemMarshal, _ := json.Marshal(input.Item)
	json.Unmarshal(itemMarshal, &buildconfig)

	buildconfig, err := p.updateSecretsAndDockerRefs(buildconfig)
	if err != nil {
		p.Log.Error("[buildconfig-restore] error modifying buildconfig: ", err)
		return nil, err
	}

	var out map[string]interface{}
	objrec, _ := json.Marshal(buildconfig)
	json.Unmarshal(objrec, &out)

	return velero.NewRestoreItemActionExecuteOutput(&unstructured.Unstructured{Object: out}), nil
}

func (p *RestorePlugin) updateSecretsAndDockerRefs(buildconfig buildv1API.BuildConfig) (buildv1API.BuildConfig, error) {
	client, err := clients.CoreClient()
	if err != nil {
		return buildconfig, err
	}

	secretList, err := client.Secrets(buildconfig.Namespace).List(metav1.ListOptions{})
	if err != nil {
		return buildconfig, err
	}

	registry := buildconfig.Annotations[common.RestoreRegistryHostname]
	if registry == "" {
		err = fmt.Errorf("failed to find restore registry annotation")
		return buildconfig, err
	}
	backupRegistry := buildconfig.Annotations[common.BackupRegistryHostname]
	if backupRegistry == "" {
		err = fmt.Errorf("failed to find backup registry annotation")
		return buildconfig, err
	}

	newCommonSpec, err := build.UpdateCommonSpec(buildconfig.Spec.CommonSpec, registry, backupRegistry, secretList, p.Log)
	if err != nil {
		return buildconfig, err
	}
	buildconfig.Spec.CommonSpec = newCommonSpec
	return buildconfig, nil
}
