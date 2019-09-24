package upstreams

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/blang/semver"
	"github.com/google/go-github/github"
	log "github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

// Github upstream representation
type Github struct {
	UpstreamBase `mapstructure:",squash"`
	// Github URL, e.g. hashicorp/terraform or helm/helm
	URL string
	// Optional: semver constraints, e.g. < 2.0.0
	// Will have no effect if the dependency does not follow Semver
	Constraints string
}

func getClient() *github.Client {
	var client *github.Client
	accessToken := os.Getenv("GITHUB_ACCESS_TOKEN")
	if accessToken != "" {
		log.Debugf("GitHub Access Token provided")
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: accessToken})
		tc := oauth2.NewClient(oauth2.NoContext, ts)
		client = github.NewClient(tc)
	} else {
		log.Warnf("No GitHub Access Token provided, might run into API limits. Set an access token with the GITHUB_ACCESS_TOKEN env var.")
		client = github.NewClient(nil)
	}
	return client
}

// LatestVersion returns the latest non-draft, non-prerelease Github Release for the given repository (depending on the Constraints if set).
//
// Authentication
//
// The Github API allows unauthenticated requests, but the API limits are very strict: https://developer.github.com/v3/#rate-limiting
//
// To authenticate your requests, use the GITHUB_ACCESS_TOKEN environment variable.
func (upstream Github) LatestVersion() (string, error) {
	log.Debugf("Using GitHub flavour")
	return latestVersion(upstream, getClient)
}

func latestVersion(upstream Github, getClient func() *github.Client) (string, error) {
	client := getClient()
	if !strings.Contains(upstream.URL, "/") {
		return "", fmt.Errorf("Invalid github repo: %v\nGithub repo should be in the form owner/repo, e.g. kubernetes/kubernetes", upstream.URL)
	}

	semverConstraints := upstream.Constraints
	if semverConstraints == "" {
		// If no range is passed, just use the broadest possible range
		semverConstraints = ">= 0.0.0"
	}
	expectedRange, err := semver.ParseRange(semverConstraints)
	if err != nil {
		return "", fmt.Errorf("Invalid semver constraints range: %v", upstream.Constraints)
	}

	splitURL := strings.Split(upstream.URL, "/")
	owner := splitURL[0]
	repo := splitURL[1]
	opt := &github.ListOptions{PerPage: 30}

	// We'll need to fetch all releases, as Github doesn't provide sorting options.
	// If we don't do that, we risk running into the case where for example:
	// - Version 1.0.0 and 2.0.0 exist
	// - A bugfix 1.0.1 gets released
	// - Now the "latest" (date-wise) release is not the highest semver, and not necessarily the one we want
	var releases []*github.RepositoryRelease
	for {
		releasesInPage, resp, err := client.Repositories.ListReleases(context.Background(), owner, repo, opt)
		if err != nil {
			return "", fmt.Errorf("Cannot list releases for repository %v/%v, error: %v", owner, repo, err)
		}
		releases = append(releases, releasesInPage...)
		// Pagination handling: if there's a next page, try it too
		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	for _, release := range releases {
		if release.TagName == nil {
			log.Debugf("Skipping release without TagName")
		}
		tag := *release.TagName
		if release.Draft != nil && *release.Draft {
			log.Debugf("Skipping draft release: %v\n", tag)
			continue
		}
		if release.Prerelease != nil && *release.Prerelease {
			log.Debugf("Skipping prerelease release: %v\n", tag)
			continue
		}

		// Try to match semver and range
		version, err := semver.Parse(strings.Trim(tag, "v"))
		if err != nil {
			log.Debugf("Error parsing version %v (%v)as semver, cannot validate semver constraints", tag, err)
		} else {
			if !expectedRange(version) {
				log.Debugf("Skipping release not matching range constraints (%v): %v\n", upstream.Constraints, tag)
				continue
			}
		}

		log.Debugf("Found latest matching release: %v\n", version)
		return version.String(), nil
	}

	// No latest version found – no versions? Only prereleases?
	return "", fmt.Errorf("No potential version found")
}