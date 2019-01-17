package healthcheck

import (
	"context"
	"time"

	pb "github.com/linkerd/linkerd2/controller/gen/public"
	"github.com/linkerd/linkerd2/pkg/version"
)

// GetServerVersion returns the Linkerd Public API server version
func GetServerVersion(apiClient pb.ApiClient) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rsp, err := apiClient.Version(ctx, &pb.Empty{})
	if err != nil {
		return "", err
	}

	return rsp.GetReleaseVersion(), nil
}

// TODO: delete
// CheckServerVersion validates whether the Linkerd Public API server's version
// matches an expected version.
func CheckServerVersion(apiClient pb.ApiClient, expectedVersion string) error {
	releaseVersion, err := GetServerVersion(apiClient)
	if err != nil {
		return err
	}

	return version.Match(expectedVersion, releaseVersion)
}
