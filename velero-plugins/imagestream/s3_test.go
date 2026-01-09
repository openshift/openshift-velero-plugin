package imagestream

import "testing"

func TestGetBucketRegion(t *testing.T) {
	type args struct {
		bucket string
	}
	tests := []struct {
		name       string
		bucket     string
		mockRegion string
		mockErr    error
		wantRegion string
		wantErr    bool
	}{
		{
			name:       "openshift-velero-plugin-s3-auto-region-test-1",
			bucket:     "openshift-velero-plugin-s3-auto-region-test-1",
			mockRegion: "us-east-1",
			mockErr:    nil,
			wantRegion: "us-east-1",
			wantErr:    false,
		},
		{
			name:       "openshift-velero-plugin-s3-auto-region-test-2",
			bucket:     "openshift-velero-plugin-s3-auto-region-test-2",
			mockRegion: "us-west-1",
			mockErr:    nil,
			wantRegion: "us-west-1",
			wantErr:    false,
		},
		{
			name:       "openshift-velero-plugin-s3-auto-region-test-3",
			bucket:     "openshift-velero-plugin-s3-auto-region-test-3",
			mockRegion: "eu-central-1",
			mockErr:    nil,
			wantRegion: "eu-central-1",
			wantErr:    false,
		},
		{
			name:       "openshift-velero-plugin-s3-auto-region-test-4",
			bucket:     "openshift-velero-plugin-s3-auto-region-test-4",
			mockRegion: "sa-east-1",
			mockErr:    nil,
			wantRegion: "sa-east-1",
			wantErr:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original and restore after test
			originalFunc := GetBucketRegionFunc
			defer func() { GetBucketRegionFunc = originalFunc }()

			// Set up mock
			GetBucketRegionFunc = func(bucket string) (string, error) {
				if bucket != tt.bucket {
					t.Errorf("GetBucketRegion called with wrong bucket: got %v, want %v", bucket, tt.bucket)
				}
				return tt.mockRegion, tt.mockErr
			}

			got, err := GetBucketRegion(tt.bucket)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBucketRegion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantRegion {
				t.Errorf("GetBucketRegion() = %v, want %v", got, tt.wantRegion)
			}
		})
	}
}
