/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package upstreams

import (
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	log "github.com/sirupsen/logrus"
)

// AMI is the Amazon Machine Image upstream
//
// See: https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/AMIs.html
type AMI struct {
	UpstreamBase `mapstructure:",squash"`
	// Either owner alias (e.g. "amazon") or owner id
	Owner string
	// Name predicate, as used in --filter
	// Supports wilcards
	Name string
}

// LatestVersion returns the latest version of an AMI.
//
// Returns the latest ami id (e.g. `ami-1234567`) from all AMIs matching the predicates, sorted by CreationDate.
//
// If images cannot be listed, or if no image matches the predicates, it will return an error instead.
//
// Authentication
//
// Authentication is provided by the standard AWS credentials use the standard `~/.aws/config` and `~/.aws/credentials` files, and support environment variables.
// See AWS documentation for more details:
// https://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/sessions.html
func (upstream AMI) LatestVersion() (string, error) {
	log.Debugf("Using AMI upstream")

	// Create a new session based on shared / env credentials
	s := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))
	svc := ec2.New(s)

	// Generate filters based on configuration
	var filters []*ec2.Filter
	filters = append(filters, &ec2.Filter{
		Name:   aws.String("name"),
		Values: []*string{aws.String(upstream.Name)},
	})

	input := &ec2.DescribeImagesInput{
		Owners:  []*string{aws.String(upstream.Owner)},
		Filters: filters,
	}

	// Do the actual API call
	result, err := svc.DescribeImages(input)
	if err != nil {
		return "", err
	}
	images := result.Images

	// Sort images by creation time, so we can return the latest
	sort.Slice(images, func(i, j int) bool { return *images[i].CreationDate > *images[j].CreationDate })
	log.Debugf("Matched AMIs:\n%s", images)

	if len(images) < 1 {
		return "", fmt.Errorf("no AMI found for upstream %s", upstream)
	}

	latestImage := images[0]
	log.Debugf("Latest AMI:\n%s", latestImage)

	return *latestImage.ImageId, nil
}
