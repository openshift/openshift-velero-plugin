package serviceaccount

import (
	"github.com/sirupsen/logrus"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	"k8s.io/apimachinery/pkg/runtime"
)

// IBAPlugin is an ItemBlock action plugin for Velero.
type IBAPlugin struct {
	Log              logrus.FieldLogger
	sccCache
}

// AppliesTo returns a velero.ResourceSelector that applies to everything.
func (p *IBAPlugin) AppliesTo() (velero.ResourceSelector, error) {
	return velero.ResourceSelector{
		IncludedResources: []string{"serviceaccounts"},
	}, nil
}

// GetRelatedItems returns a list of SCCs related to this ServiceAccount
func (p *IBAPlugin) GetRelatedItems(item runtime.Unstructured, backup *v1.Backup) ([]velero.ResourceIdentifier, error) {
	p.Log.Info("[serviceaccount-iba] Entering ServiceAccount ItemBlock plugin")
	return sccsForSA(p.Log, item, backup, p.sccCache)
}

// This won't be called but is needed to implement interface
func (p *IBAPlugin) Name() string {
	return "serviceaccount-iba"
}
