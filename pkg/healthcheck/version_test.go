package healthcheck

import (
	"errors"
	"fmt"
	"testing"

	"github.com/linkerd/linkerd2/controller/api/public"
	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/version"
)

func TestCheckServerVersion(t *testing.T) {
	t.Run("Passes when server version matches", func(t *testing.T) {
		apiClient := createMockPublicAPI(version.Version)
		err := CheckServerVersion(apiClient, version.Version)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	})

	t.Run("Fails when server version does not match", func(t *testing.T) {
		channel := "channel"
		expected := fmt.Sprintf("%s-%s", channel, version.Version)
		latest := expected + "latest"
		expectedErr := fmt.Errorf("is running version %s but the latest %s version is %s", version.Version+"latest", channel, version.Version)
		apiClient := createMockPublicAPI(latest)

		err := CheckServerVersion(apiClient, expected)
		if err == nil || err.Error() != expectedErr.Error() {
			t.Fatalf("Expected \"%s\", got \"%s\"", expectedErr, err)
		}
	})
}

func createMockPublicAPI(version string) *public.MockAPIClient {
	return &public.MockAPIClient{
		VersionInfoToReturn: &pb.VersionInfo{
			ReleaseVersion: version,
		},
	}
}

func TestGetServerVersion(t *testing.T) {
	t.Run("Returns existing version from server", func(t *testing.T) {
		expectedServerVersion := "1.2.3"
		mockClient := &public.MockAPIClient{}
		mockClient.VersionInfoToReturn = &pb.VersionInfo{
			ReleaseVersion: expectedServerVersion,
		}

		version, err := GetServerVersion(mockClient)
		if err != nil {
			t.Fatalf("GetServerVersion returned unexpected error: %s", err)
		}

		if version != expectedServerVersion {
			t.Fatalf("Expected server version to be [%s], was [%s]",
				expectedServerVersion, version)
		}
	})

	t.Run("Returns an error when cannot get server version", func(t *testing.T) {
		mockClient := &public.MockAPIClient{}
		mockClient.ErrorToReturn = errors.New("expected")

		_, err := GetServerVersion(mockClient)
		if err != mockClient.ErrorToReturn {
			t.Fatalf("GetServerVersion returned unexpected error: %s", err)
		}
	})
}
