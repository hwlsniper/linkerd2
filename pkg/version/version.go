package version

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

type channelVersion struct {
	channel string
	version string
}

var undefinedChannel = "dev"
var undefinedVersion = "undefined"
var undefinedChannelVersion = channelVersion{"dev", "undefined"}.String()

// Version is updated automatically as part of the build process
//
// DO NOT EDIT
var Version = undefinedChannelVersion

// Channels provides an interface to interact with a set of release channels
type Channels struct {
	array []channelVersion
}

const (
	versionCheckURL = "https://versioncheck.linkerd.io/version.json?version=%s&uuid=%s&source=%s"
)

func init() {
	// Use `$LINKERD_CONTAINER_VERSION_OVERRIDE` as the version only if the
	// version wasn't set at link time to minimize the chance of using it
	// unintentionally. This mechanism allows the version to be bound at
	// container build time instead of at executable link time to improve
	// incremental rebuild efficiency.
	if Version == undefinedChannelVersion {
		override := os.Getenv("LINKERD_CONTAINER_VERSION_OVERRIDE")
		if override != "" {
			Version = override
		}
	}
}

// TODO: delete
// CheckClientVersion validates whether the Linkerd Public API client's version
// matches an expected version.
func CheckClientVersion(expectedVersion string) error {
	return Match(expectedVersion, Version)
}

// GetLatestVersions performs an online request to check for the latest Linkerd
// release channels.
func GetLatestVersions(uuid string, source string) (Channels, error) {
	url := fmt.Sprintf(versionCheckURL, Version, uuid, source)
	return getLatestVersions(http.DefaultClient, url, uuid, source)
}

func getLatestVersions(client *http.Client, url string, uuid string, source string) (Channels, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Channels{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rsp, err := client.Do(req.WithContext(ctx))
	if err != nil {
		return Channels{}, err
	}
	defer rsp.Body.Close()

	if rsp.StatusCode != 200 {
		return Channels{}, fmt.Errorf("Unexpected versioncheck response: %s", rsp.Status)
	}

	bytes, err := ioutil.ReadAll(rsp.Body)
	if err != nil {
		return Channels{}, err
	}

	var versionRsp map[string]string
	err = json.Unmarshal(bytes, &versionRsp)
	if err != nil {
		return Channels{}, err
	}

	channels := Channels{}
	for c, v := range versionRsp {
		channels.array = append(channels.array, channelVersion{c, v})
	}

	sort.Sort(byCV(channels.array))

	return channels, nil
}

// Match compares two versions and returns success if they match, or an error
// with a contextual message if they do not.
func Match(expectedVersion, actualVersion string) error {
	if expectedVersion == "" {
		return errors.New("expected version is empty")
	} else if actualVersion == "" {
		return errors.New("actual version is empty")
	} else if actualVersion == expectedVersion {
		return nil
	}

	actual, err := parseChannelVersion(actualVersion)
	if err != nil {
		return fmt.Errorf("failed to parse actual version: %s", err)
	}
	expected, err := parseChannelVersion(expectedVersion)
	if err != nil {
		return fmt.Errorf("failed to parse expected version: %s", err)
	}

	if actual.channel != expected.channel {
		return fmt.Errorf("mismatched channels: running %s but retrieved %s",
			actual, expected)
	}

	return fmt.Errorf("is running version %s but the latest %s version is %s",
		actual.version, actual.channel, expected.version)
}

func (cv channelVersion) String() string {
	return fmt.Sprintf("%s-%s", cv.channel, cv.version)
}

func parseChannelVersion(cv string) (channelVersion, error) {
	if parts := strings.SplitN(cv, "-", 2); len(parts) == 2 {
		return channelVersion{
			channel: parts[0],
			version: parts[1],
		}, nil
	}
	return channelVersion{}, fmt.Errorf("unsupported version format: %s", cv)
}

func NewChannels(channels []string) (Channels, error) {
	c := Channels{}
	for _, channel := range channels {
		cv, err := parseChannelVersion(channel)
		if err != nil {
			return Channels{}, err
		}

		c.array = append(c.array, cv)
	}

	return c, nil
}

func (c Channels) Match(actualVersion string) error {
	if actualVersion == "" {
		return errors.New("actual version is empty")
	}

	actual, err := parseChannelVersion(actualVersion)
	if err != nil {
		return fmt.Errorf("failed to parse actual version: %s", err)
	}

	for _, cv := range c.array {
		if cv.channel == actual.channel {
			return Match(cv.String(), actual.String())
		}
	}

	return fmt.Errorf("unsupported version channel: %s", actualVersion)
}

type byCV []channelVersion

func (b byCV) Len() int {
	return len(b)
}

func (b byCV) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}

func (b byCV) Less(i, j int) bool {
	if b[i].channel == "" {
		return true
	}
	if b[j].channel == "" {
		return false
	}

	return b[i].channel < b[j].channel
}
