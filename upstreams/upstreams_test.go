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
	"testing"

	"gopkg.in/yaml.v3"
)

func TestUpstreamBaseLatestVersion(t *testing.T) {
	var u UpstreamBase

	input := []byte("flavour: dummy")

	err := yaml.Unmarshal(input, &u)
	if err != nil {
		t.Errorf("Failed to deserialise valid yaml:\n%s\nError: %v", input, err)
	}

	_, err = u.LatestVersion()
	if err == nil {
		t.Errorf("LatestVersion on UpstreamBase should return an error")
	}
}
