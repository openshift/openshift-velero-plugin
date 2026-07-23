package imagecopy

import (
	"testing"

	"github.com/containers/image/v5/copy"
	"github.com/go-logr/logr"
	imagev1API "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	testInternalRegistry  = "image-registry.openshift-image-registry.svc:5000"
	testMigrationRegistry = "migration-registry.example.com:5000"
	testDigest            = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
)

type copyCall struct {
	src  string
	dest string
}

// stubCopyImage replaces copyImageFn for the duration of the test, recording
// the src/dest paths instead of talking to a registry.
func stubCopyImage(t *testing.T) *[]copyCall {
	t.Helper()
	calls := &[]copyCall{}
	orig := copyImageFn
	copyImageFn = func(_ logr.Logger, src, dest string, _ *copy.Options) ([]byte, error) {
		*calls = append(*calls, copyCall{src: src, dest: dest})
		return []byte(`{"schemaVersion":2}`), nil
	}
	t.Cleanup(func() { copyImageFn = orig })
	return calls
}

func newImageStream(namespace, name, dockerImageReference string) imagev1API.ImageStream {
	return imagev1API.ImageStream{
		ObjectMeta: metav1.ObjectMeta{Namespace: namespace, Name: name},
		Status: imagev1API.ImageStreamStatus{
			Tags: []imagev1API.NamedTagEventList{
				{
					Tag: "v1",
					Items: []imagev1API.TagEvent{
						{
							DockerImageReference: dockerImageReference,
							Image:                testDigest,
						},
					},
				},
			},
		},
	}
}

func defaultOptions() CopyLocalImageStreamImagesOptions {
	return CopyLocalImageStreamImagesOptions{
		InternalRegistryPath: testInternalRegistry,
		SrcRegistry:          testInternalRegistry,
		DestRegistry:         testMigrationRegistry,
		DestNamespace:        "namespace-int",
		Log:                  logr.Discard(),
	}
}

// Images promoted across namespaces (e.g. `oc tag namespace-dev/imagefoo:v1
// namespace-int/imagefoo:v1`) can leave a status item whose
// DockerImageReference points at the *source* namespace's repository. The
// internal registry only serves a digest from a repository whose imagestream
// still references it, so once the source tag is deleted/pruned that path
// returns "manifest unknown". The digest is always a member of the stream
// being copied, so the copy source must be the stream's own repository.
// https://github.com/openshift/openshift-velero-plugin/issues/443
func TestCopyLocalImageStreamImagesCrossNamespaceReference(t *testing.T) {
	calls := stubCopyImage(t)

	is := newImageStream("namespace-int", "imagefoo",
		testInternalRegistry+"/namespace-dev/imagefoo@"+testDigest)

	if err := CopyLocalImageStreamImages(is, defaultOptions()); err != nil {
		t.Fatalf("CopyLocalImageStreamImages returned error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 copy call, got %d", len(*calls))
	}
	wantSrc := "docker://" + testInternalRegistry + "/namespace-int/imagefoo@" + testDigest
	if got := (*calls)[0].src; got != wantSrc {
		t.Errorf("src path must use the imagestream's own repository\n got: %s\nwant: %s", got, wantSrc)
	}
	wantDest := "docker://" + testMigrationRegistry + "/namespace-int/imagefoo:v1"
	if got := (*calls)[0].dest; got != wantDest {
		t.Errorf("unexpected dest path\n got: %s\nwant: %s", got, wantDest)
	}
}

// Same-namespace references must keep producing the exact same paths as
// before (the common case).
func TestCopyLocalImageStreamImagesSameNamespaceReference(t *testing.T) {
	calls := stubCopyImage(t)

	is := newImageStream("namespace-int", "imagefoo",
		testInternalRegistry+"/namespace-int/imagefoo@"+testDigest)

	if err := CopyLocalImageStreamImages(is, defaultOptions()); err != nil {
		t.Fatalf("CopyLocalImageStreamImages returned error: %v", err)
	}
	if len(*calls) != 1 {
		t.Fatalf("expected 1 copy call, got %d", len(*calls))
	}
	wantSrc := "docker://" + testInternalRegistry + "/namespace-int/imagefoo@" + testDigest
	if got := (*calls)[0].src; got != wantSrc {
		t.Errorf("unexpected src path\n got: %s\nwant: %s", got, wantSrc)
	}
}

// Images not hosted in the internal registry are not local and must not be copied.
func TestCopyLocalImageStreamImagesExternalImageSkipped(t *testing.T) {
	calls := stubCopyImage(t)

	is := newImageStream("namespace-int", "imagefoo",
		"quay.io/libpod/busybox@"+testDigest)

	if err := CopyLocalImageStreamImages(is, defaultOptions()); err != nil {
		t.Fatalf("CopyLocalImageStreamImages returned error: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("expected no copy calls for external image, got %d: %+v", len(*calls), *calls)
	}
}

// Verbatim reference tags (`oc tag --reference`) produce tag events with an
// internal-registry DockerImageReference but no image digest. There is nothing
// content-addressed to copy, so the item must be skipped rather than producing
// a malformed "<repo>@" source reference.
func TestCopyLocalImageStreamImagesReferenceTagWithoutDigestSkipped(t *testing.T) {
	calls := stubCopyImage(t)

	is := newImageStream("namespace-int", "imagefoo",
		testInternalRegistry+"/namespace-dev/imagefoo@"+testDigest)
	is.Status.Tags[0].Items[0].Image = ""

	if err := CopyLocalImageStreamImages(is, defaultOptions()); err != nil {
		t.Fatalf("CopyLocalImageStreamImages returned error: %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("expected no copy calls for digest-less reference tag, got %d: %+v", len(*calls), *calls)
	}
}

// Every history item of a tag is copied, each from the stream's own
// repository under its own digest.
func TestCopyLocalImageStreamImagesTagHistory(t *testing.T) {
	calls := stubCopyImage(t)

	const olderDigest = "sha256:fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	is := newImageStream("namespace-int", "imagefoo",
		testInternalRegistry+"/namespace-dev/imagefoo@"+testDigest)
	is.Status.Tags[0].Items = append(is.Status.Tags[0].Items, imagev1API.TagEvent{
		DockerImageReference: testInternalRegistry + "/namespace-dev/imagefoo@" + olderDigest,
		Image:                olderDigest,
	})

	if err := CopyLocalImageStreamImages(is, defaultOptions()); err != nil {
		t.Fatalf("CopyLocalImageStreamImages returned error: %v", err)
	}
	if len(*calls) != 2 {
		t.Fatalf("expected 2 copy calls, got %d", len(*calls))
	}
	// items are iterated in reverse so the most recent is copied last
	wantSrcs := []string{
		"docker://" + testInternalRegistry + "/namespace-int/imagefoo@" + olderDigest,
		"docker://" + testInternalRegistry + "/namespace-int/imagefoo@" + testDigest,
	}
	for i, want := range wantSrcs {
		if got := (*calls)[i].src; got != want {
			t.Errorf("call %d: src path must use the imagestream's own repository\n got: %s\nwant: %s", i, got, want)
		}
	}
}
