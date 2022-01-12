package common

import (
	"github.com/vmware-tanzu/velero/pkg/util/collections"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Snippets from https://github.com/vmware-tanzu/velero/blob/main/internal/hook/item_hook_handler.go

type ResourceHookSelector struct {
	Namespaces    *collections.IncludesExcludes
	Resources     *collections.IncludesExcludes
	LabelSelector labels.Selector
}

func (r ResourceHookSelector) ApplicableTo(groupResource schema.GroupResource, namespace string, labels labels.Set) bool {
	if r.Namespaces != nil && !r.Namespaces.ShouldInclude(namespace) {
		return false
	}
	if r.Resources != nil && !r.Resources.ShouldInclude(groupResource.String()) {
		return false
	}
	if r.LabelSelector != nil && !r.LabelSelector.Matches(labels) {
		return false
	}
	return true
}