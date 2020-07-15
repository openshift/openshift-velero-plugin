package imagestream

import(
	"testing"
	"github.com/containers/image/v5/signature"
        imagev1API "github.com/openshift/api/image/v1"
	"reflect"
	"github.com/containers/image/v5/types"
	"github.com/stretchr/testify/require"
)


/*
func TestCopyImage(t *testing.T) {
	tests := map[string]struct {
		src, dest string
		sourceCtx, destinationCtx *types.SystemContext
                exp []byte
		expErr error
        }{
		"1": {src: "", dest: "", sourceCtx: *types.SystemContext{}, destinationCtx: *types.SystemContext{}, exp: []byte{}, expErr: nil},
        }

        for name, tc := range tests {
                t.Run(name, func(t *testing.T) {
			got, err := copyImage(src)
			if tc.expErr == nil && got != tc.exp {
                                t.Fatalf("expected: %v, got: %v", tc.exp, got)
                        }
			if tc.expErr != nil && err == nil {
				t.Fatalf("expected error, got no error")
			}
                })
        }
}
*/

func TestGetPolicyContext(t *testing.T) {
	exp, expErr := signature.NewPolicyContext(&signature.Policy{Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()}})
	actual, actualErr := getPolicyContext()
	if !reflect.DeepEqual(actualErr, expErr) {
                t.Fatalf("expected err: %v, got: %v", expErr, actualErr)
        }
	if !reflect.DeepEqual(actual, exp) {
                t.Fatalf("expected: %v, got: %v", exp, actual)
        }
}

/*
func TestInternalRegistrySystemContext(t *testing.T) {
	actual, err := internalRegistrySystemContext()
	require.NoError(t, err)
	exp := &types.SystemContext{
                DockerDaemonInsecureSkipTLSVerify: true,
                DockerInsecureSkipTLSVerify:       types.OptionalBoolTrue,
                DockerDisableDestSchema1MIMETypes: true,
                DockerAuthConfig: &types.DockerAuthConfig{
                        Username: "ignored",
                        Password: "",
                },
        }
	if !reflect.DeepEqual(actual, exp) {
		t.Fatalf("expected: %v, got: %v", exp, actual)
	}
}
*/

func TestMigrationRegistrySystemContext(t *testing.T) {
	actual, err := migrationRegistrySystemContext()
	require.NoError(t, err)
	exp := &types.SystemContext{
                DockerDaemonInsecureSkipTLSVerify: true,
                DockerInsecureSkipTLSVerify:       types.OptionalBoolTrue,
                DockerDisableDestSchema1MIMETypes: true,
        }
        if !reflect.DeepEqual(actual, exp) {
		t.Fatalf("expected: %v, got: %v", exp, actual)
        }
}

func TestFindSpecTag(t *testing.T) {
	tests := map[string]struct {
		tags []imagev1API.TagReference
		name string
		exp  *imagev1API.TagReference
	}{
		"1": {
			tags: []imagev1API.TagReference{
				imagev1API.TagReference{Name: "name1"},
				imagev1API.TagReference{Name: "name2"},
				imagev1API.TagReference{Name: "name3"},
			},
			name: "name4",
			exp: nil,
		},

		"2": {
			tags: []imagev1API.TagReference{
				imagev1API.TagReference{Name: "name1"},
				imagev1API.TagReference{Name: "name2"},
				imagev1API.TagReference{Name: "name3"},
			},
			name: "name2",
			exp: &imagev1API.TagReference{Name: "name2"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			out := findSpecTag(tc.tags, tc.name)
			if !reflect.DeepEqual(out, tc.exp) {
                                t.Fatalf("expected: %v, got: %v", tc.exp, out)
                        }
		})
	}
}
