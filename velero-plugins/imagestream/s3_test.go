package imagestream

import "testing"

func TestGetBucketRegion(t *testing.T) {
	type args struct {
		bucket string
	}
	tests := []struct {
		name    string
		bucket  string
		want    string
		wantErr bool
	}{
		{
		  	name: "openshift-velero-plugin-s3-auto-region-test-1",
				bucket: "openshift-velero-plugin-s3-auto-region-test-1",
				want: "us-east-1",
				wantErr: false,
		},
		{
		  	name: "openshift-velero-plugin-s3-auto-region-test-2",
				bucket: "openshift-velero-plugin-s3-auto-region-test-2",
				want: "us-west-1",
				wantErr: false,
		},
		{
		  	name: "openshift-velero-plugin-s3-auto-region-test-3",
				bucket: "openshift-velero-plugin-s3-auto-region-test-3",
				want: "eu-central-1",
				wantErr: false,
		},
		{
		  	name: "openshift-velero-plugin-s3-auto-region-test-4",
				bucket: "openshift-velero-plugin-s3-auto-region-test-4",
				want: "sa-east-1",
				wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetBucketRegion(tt.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBucketRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetBucketRegion() = %v, want %v", got, tt.want)
			}
		})
	}
}
