package imagestream

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestRestorePluginAppliesTo(t *testing.T) {
	BackupPlugin := &BackupPlugin{Log: test.NewLogger()}
	actual, err := BackupPlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"imagestreams"}}, actual)
}

func TestRestorePlugin_Execute(t *testing.T) {
	restorePlugin := &RestorePlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		im imagev1API.ImageStream
		test bool
	}{
		"ImagestreamLogicRun": {
			im: imagev1API.ImageStream{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.MigrationRegistry: "migReg",
						common.BackupRegistryHostname: "backupReg",
						common.RestoreRegistryHostname: "restoreReg",
					},
				},
				Spec:   imagev1API.ImageStreamSpec{},
				Status: imagev1API.ImageStreamStatus{},
			},
			test: true,
		},
		"NoImagestreamLogicRun": {
			im: imagev1API.ImageStream{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.MigrationRegistry: "",
						common.BackupRegistryHostname: "backupReg",
						common.RestoreRegistryHostname: "restoreReg",
					},
				},
				Spec:   imagev1API.ImageStreamSpec{},
				Status: imagev1API.ImageStreamStatus{},
			},
			test: false,
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			imRec, _ := json.Marshal(tc.im)
			json.Unmarshal(imRec, &out)
			item.SetUnstructuredContent(out)

			input := velero.RestoreItemActionExecuteInput{Item: &item}

			output, err := restorePlugin.Execute(&input)

			imagestream := imagev1API.ImageStream{}
			itemMarshal, _ := json.Marshal(output)
			json.Unmarshal(itemMarshal, &imagestream)

			if err != nil && tc.test == true {
				t.Fatalf("Logic didnt run when it should.")
			}
			if err == nil && tc.test == false {
				t.Fatalf("Logic ran but should not.")
			}
		})
	}
}
