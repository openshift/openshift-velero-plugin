package imagestream

import (
	"encoding/json"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/common"
	"github.com/konveyor/openshift-velero-plugin/velero-plugins/util/test"
	imagev1API "github.com/openshift/api/image/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	"github.com/vmware-tanzu/velero/pkg/plugin/velero"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"testing"
)

func TestBackupPlugin_AppliesToPluginAppliesTo(t *testing.T) {
	BackupPlugin := &BackupPlugin{Log: test.NewLogger()}
	actual, err := BackupPlugin.AppliesTo()
	require.NoError(t, err)
	assert.Equal(t, velero.ResourceSelector{IncludedResources: []string{"imagestreams"}}, actual)
}

func TestBackupPlugin_ExecutePlugin_Execute(t *testing.T) {
	BackupPlugin := &BackupPlugin{Log: test.NewLogger()}

	testcase := map[string]struct {
		im       imagev1API.ImageStream
		backup   v1.Backup
	}{
		"ImagestreamLogicRun": {
			im: imagev1API.ImageStream{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname: "backupReg",
						common.MigrationRegistry:      "migReg",
					},
				},
				Spec:   imagev1API.ImageStreamSpec{},
				Status: imagev1API.ImageStreamStatus{},
			},
			backup: v1.Backup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       v1.BackupSpec{},
				Status:     v1.BackupStatus{},
			},
		},
		"NoImagestreamLogicRun": {
			im: imagev1API.ImageStream{
				TypeMeta: metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						common.BackupRegistryHostname: "backupReg",
						common.MigrationRegistry:      "",
					},
				},
				Spec:   imagev1API.ImageStreamSpec{},
				Status: imagev1API.ImageStreamStatus{},
			},
			backup: v1.Backup{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       v1.BackupSpec{},
				Status:     v1.BackupStatus{},
			},
		},
	}

	for i, tc := range testcase {
		t.Run(string(i), func(t *testing.T) {
			var out map[string]interface{}
			item := unstructured.Unstructured{}
			imRec, _ := json.Marshal(tc.im)
			json.Unmarshal(imRec, &out)
			item.SetUnstructuredContent(out)

			output, _, err := BackupPlugin.Execute(&item, &tc.backup)

			imagestream := imagev1API.ImageStream{}
			itemMarshal, _ := json.Marshal(output)
			json.Unmarshal(itemMarshal, &imagestream)

			if err == nil && imagestream.Annotations[common.MigrationRegistry] == "" {
				t.Fatalf("Logic ran when it should not.")
			}
			if err != nil && imagestream.Annotations[common.MigrationRegistry] == "migReg" {
				t.Fatalf("Logic didnt run when it should.")
			}
		})
	}
}
