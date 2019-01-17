package version

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCheckClientVersion(t *testing.T) {
	t.Run("Passes when client version matches", func(t *testing.T) {
		err := CheckClientVersion(Version)
		if err != nil {
			t.Fatalf("Unexpected error: %s", err)
		}
	})

	t.Run("Fails when client version does not match", func(t *testing.T) {
		latest := channelVersion{undefinedChannel, "latest"}
		expectedErr := fmt.Errorf("is running version %s but the latest %s version is %s", undefinedVersion, undefinedChannel, latest.version)

		err := CheckClientVersion(latest.String())
		if err == nil || err.Error() != expectedErr.Error() {
			t.Fatalf("Expected \"%s\", got \"%s\"", expectedErr, err)
		}
	})
}

func TestGetLatestVersions(t *testing.T) {
	testCases := []struct {
		resp   interface{}
		err    error
		latest Channels
	}{
		{
			map[string]string{
				undefinedChannel: undefinedVersion,
				"foo":            "foo-1.2.3",
				"version":        "stable-2.1.0",
			},
			nil,
			Channels{
				[]channelVersion{
					{undefinedChannel, undefinedVersion},
					{"foo", "foo-1.2.3"},
					{"version", "stable-2.1.0"},
				},
			},
		},
		{
			"bad response",
			fmt.Errorf("json: cannot unmarshal string into Go value of type map[string]string"),
			Channels{},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d GetLatestVersions(%s, %s)", i, tc.err, tc.latest), func(t *testing.T) {
			j, err := json.Marshal(tc.resp)
			if err != nil {
				t.Fatalf("JSON marshal failed with: %s", err)
			}

			ts := httptest.NewServer(http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					w.Write(j)
				}),
			)
			defer ts.Close()

			latest, err := getLatestVersions(ts.Client(), ts.URL, "uuid", "source")
			if (err == nil && tc.err != nil) ||
				(err != nil && tc.err == nil) ||
				((err != nil && tc.err != nil) && (err.Error() != tc.err.Error())) {
				t.Fatalf("Expected \"%s\", got \"%s\"", tc.err, err)
			}

			if !reflect.DeepEqual(latest, tc.latest) {
				t.Fatalf("Expected latest versions \"%s\", got \"%s\"", tc.latest, latest)
			}
		})
	}
}

func TestChannelsMatch(t *testing.T) {
	testCases := []struct {
		actualVersion string
		channels      Channels
		err           error
	}{
		{
			undefinedChannelVersion,
			Channels{
				[]channelVersion{
					{"version", "stable-2.1.0"},
					{"foo", "foo-1.2.3"},
					{undefinedChannel, undefinedVersion},
				},
			},
			nil,
		},
		{
			channelVersion{undefinedChannel, "older"}.String(),
			Channels{
				[]channelVersion{
					{"version", "stable-2.1.0"},
					{"foo", "foo-1.2.3"},
					{undefinedChannel, "latest"},
				},
			},
			errors.New("is running version older but the latest dev version is latest"),
		},
		{
			"unsupported-version-channel",
			Channels{
				[]channelVersion{
					{"version", "stable-2.1.0"},
					{"foo", "foo-1.2.3"},
					{"bar", "bar-3.2.1"},
				},
			},
			fmt.Errorf("unsupported version channel: %s", "unsupported-version-channel"),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d ChannelsMatch(%s, %s)", i, tc.actualVersion, tc.err), func(t *testing.T) {

			err := tc.channels.Match(tc.actualVersion)
			if (err == nil && tc.err != nil) ||
				(err != nil && tc.err == nil) ||
				((err != nil && tc.err != nil) && (err.Error() != tc.err.Error())) {
				t.Fatalf("Expected \"%s\", got \"%s\"", tc.err, err)
			}
		})
	}
}

func TestMatch(t *testing.T) {
	testCases := []struct {
		expected string
		actual   string
		err      error
	}{
		{"dev-foo", "dev-foo", nil},
		{"dev-foo-bar", "dev-foo-bar", nil},
		{"dev-foo-bar", "dev-foo-baz", errors.New("is running version foo-baz but the latest dev version is foo-bar")},
		{"dev-foo", "dev-bar", errors.New("is running version bar but the latest dev version is foo")},
		{"dev-foo", "git-foo", errors.New("mismatched channels: running git-foo but retrieved dev-foo")},
		{"badformat", "dev-foo", errors.New("failed to parse expected version: unsupported version format: badformat")},
		{"dev-foo", "badformat", errors.New("failed to parse actual version: unsupported version format: badformat")},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("test %d Match(%s, %s)", i, tc.expected, tc.actual), func(t *testing.T) {
			err := Match(tc.expected, tc.actual)
			if (err == nil && tc.err != nil) ||
				(err != nil && tc.err == nil) ||
				((err != nil && tc.err != nil) && (err.Error() != tc.err.Error())) {
				t.Fatalf("Expected \"%s\", got \"%s\"", tc.err, err)
			}
		})
	}
}
