/*
Copyright 2019 The Kubernetes Authors.

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

package data

import (
	"testing"
)

const (
	dbFile = "testdata/boltdb/db"
)

func TestListKeySummariesFilters(t *testing.T) {
	cases := []struct {
		name         string
		filters      []Filter
		expectedKeys []string
	}{
		{
			name:         "nofilters",
			filters:      []Filter{},
			expectedKeys: []string{"/registry/jobs/default/pi", "/registry/namespaces/default", "/registry/pods/default/pi-dqtsw", "compact_rev_key"},
		},
		{
			name:         "prefixfilter",
			filters:      []Filter{NewPrefixFilter("/registry/jobs")},
			expectedKeys: []string{"/registry/jobs/default/pi"},
		},
		{
			name:         "namespacefilter",
			filters:      []Filter{mustBuildFilter(&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"})},
			expectedKeys: []string{"/registry/jobs/default/pi", "/registry/pods/default/pi-dqtsw"},
		},
		{
			name:         "allfilters",
			filters:      []Filter{NewPrefixFilter("/registry/jobs"), mustBuildFilter(&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"})},
			expectedKeys: []string{"/registry/jobs/default/pi"},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			missingKeys := map[string]struct{}{}
			for _, key := range tt.expectedKeys {
				missingKeys[key] = struct{}{}
			}
			unexpectedKeys := map[string]struct{}{}
			results, err := ListKeySummaries(dbFile, tt.filters, ProjectEverything)
			if err != nil {
				t.Fatal(err)
			}
			for _, result := range results {
				if _, ok := missingKeys[result.Key]; ok {
					delete(missingKeys, result.Key)
				} else {
					unexpectedKeys[result.Key] = struct{}{}
				}
			}
			if len(unexpectedKeys) != 0 {
				t.Errorf("got %d unexpected keys: %v, expected none", len(unexpectedKeys), unexpectedKeys)
			}
			if len(missingKeys) != 0 {
				t.Errorf("got %d missing keys: %v, expected none", len(missingKeys), missingKeys)
			}
		})
	}
}

func TestParseFilters(t *testing.T) {
	cases := []struct {
		name      string
		rawFilter string
		expected  []*FieldConstraint
	}{
		{
			name:      "namespace-equals",
			rawFilter: ".Value.metadata.namespace=default",
			expected:  []*FieldConstraint{&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"}},
		},
		{
			name:      "2-filters",
			rawFilter: ".Value.metadata.namespace=default,.Value.metadata.name=example",
			expected: []*FieldConstraint{
				&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"},
				&FieldConstraint{lhs: ".Value.metadata.name", op: Equals, rhs: "example"},
			},
		},
		{
			name:      "whitespace",
			rawFilter: " .Value.metadata.namespace=default\t, .Value.metadata.name=example\n",
			expected: []*FieldConstraint{
				&FieldConstraint{lhs: ".Value.metadata.namespace", op: Equals, rhs: "default"},
				&FieldConstraint{lhs: ".Value.metadata.name", op: Equals, rhs: "example"},
			},
		},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			filters, err := ParseFilters(tt.rawFilter)
			if err != nil {
				t.Fatal(err)
			}
			unexpected := map[FieldConstraint]struct{}{}
			missing := map[FieldConstraint]struct{}{}

			for _, expected := range tt.expected {
				missing[*expected] = struct{}{}
			}

			for _, filter := range filters {
				fc := filter.(*FieldFilter).FieldConstraint
				if _, ok := missing[*fc]; ok {
					delete(missing, *fc)
				} else {
					unexpected[*fc] = struct{}{}
				}
			}
			if len(unexpected) != 0 {
				t.Errorf("got %d unexpected filters: %#+v, expected none", len(unexpected), unexpected)
			}
			if len(missing) != 0 {
				t.Errorf("got %d missing filters: %#+v, expected none", len(missing), missing)
			}
		})
	}
}

func mustBuildFilter(fc *FieldConstraint) Filter {
	filter, err := fc.BuildFilter()
	if err != nil {
		panic(err)
	}
	return filter
}