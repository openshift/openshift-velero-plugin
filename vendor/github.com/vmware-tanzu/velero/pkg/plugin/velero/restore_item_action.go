/*
Copyright 2017, 2019 the Velero contributors.

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

package velero

import (
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	api "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
)

// RestoreItemAction is an actor that performs an operation on an individual item being restored.
type RestoreItemAction interface {
	// AppliesTo returns information about which resources this action should be invoked for.
	// A RestoreItemAction's Execute function will only be invoked on items that match the returned
	// selector. A zero-valued ResourceSelector matches all resources.
	AppliesTo() (ResourceSelector, error)

	// Execute allows the ItemAction to perform arbitrary logic with the item being restored,
	// including mutating the item itself prior to restore. The item (unmodified or modified)
	// should be returned, along with an optional slice of ResourceIdentifiers specifying additional
	// related items that should be restored, a warning (which will be logged but will not prevent
	// the item from being restored) or error (which will be logged and will prevent the item
	// from being restored) if applicable.
	Execute(input *RestoreItemActionExecuteInput) (*RestoreItemActionExecuteOutput, error)

	// AreAdditionalItemsReady allows the ItemAction to communicate whether the passed-in
	// slice of AdditionalItems (previously returned by Execute())
	// are ready. Returns true if all items are ready, and false
	// otherwise. The second return value is an error string if an
	// error occurred.
	AreAdditionalItemsReady(restore *api.Restore, additionalItems []ResourceIdentifier) (bool, error)
}

// RestoreItemActionExecuteInput contains the input parameters for the ItemAction's Execute function.
type RestoreItemActionExecuteInput struct {
	// Item is the item being restored. It is likely different from the pristine backed up version
	// (metadata reset, changed by various restore item action plugins, etc.).
	Item runtime.Unstructured
	// ItemFromBackup is the item taken from the pristine backed up version of resource.
	ItemFromBackup runtime.Unstructured
	// Restore is the representation of the restore resource processed by Velero.
	Restore *api.Restore
}

// RestoreItemActionExecuteOutput contains the output variables for the ItemAction's Execution function.
type RestoreItemActionExecuteOutput struct {
	// UpdatedItem is the item being restored mutated by ItemAction.
	UpdatedItem runtime.Unstructured

	// AdditionalItems is a list of additional related items that should
	// be restored.
	AdditionalItems []ResourceIdentifier

	// SkipRestore tells velero to stop executing further actions
	// on this item, and skip the restore step. When this field's
	// value is true, AdditionalItems will be ignored.
	SkipRestore bool

	// WaitForAdditionalItems determines whether velero will wait
	// until AreAdditionalItemsReady returns true before restoring
	// this item. If this field's value is true, then after restoring
	// the returned AdditionalItems, velero will not restore this item
	// until AreAdditionalItemsReady returns true or the timeout is
	// reached. Otherwise, AreAdditionalItemsReady is not called.
	WaitForAdditionalItems bool

	// AdditionalItemsReadyTimeout will override serverConfig.additionalItemsReadyTimeout
	// if specified. This value specifies how long velero will wait
	// for additional items to be ready before moving on.
	AdditionalItemsReadyTimeout time.Duration
}

// NewRestoreItemActionExecuteOutput creates a new RestoreItemActionExecuteOutput
func NewRestoreItemActionExecuteOutput(item runtime.Unstructured) *RestoreItemActionExecuteOutput {
	return &RestoreItemActionExecuteOutput{
		UpdatedItem: item,
	}
}

// WithoutRestore returns SkipRestore for RestoreItemActionExecuteOutput
func (r *RestoreItemActionExecuteOutput) WithoutRestore() *RestoreItemActionExecuteOutput {
	r.SkipRestore = true
	return r
}

// WithItemsWait returns RestoreItemActionExecuteOutput with WaitForAdditionalItems set to true.
func (r *RestoreItemActionExecuteOutput) WithItemsWait() *RestoreItemActionExecuteOutput {
	r.WaitForAdditionalItems = true
	return r
}
