/*
Copyright 2017 Kinvolk GmbH

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

package utils

import (
	"testing"
)

func TestIsSemVerOrNewer(t *testing.T) {
	testCases := []struct {
		keyVersion     string
		conVersion     string
		expectedResult bool
	}{
		{
			keyVersion:     "1.8.0",
			conVersion:     "1.8.0",
			expectedResult: true,
		},
		{
			keyVersion:     "1.8.1",
			conVersion:     "1.8.0",
			expectedResult: true,
		},
		{
			keyVersion:     "1.8.0",
			conVersion:     "1.8.1",
			expectedResult: false,
		},
	}

	for i, tc := range testCases {
		outResult := IsSemVerOrNewer(tc.keyVersion, tc.conVersion)
		if tc.expectedResult != outResult {
			t.Fatalf("Failure at test case %d: expected %v, got %v\n", i, tc.expectedResult, outResult)
		}
	}
}

func TestIsSemVer(t *testing.T) {
	testCases := []struct {
		keyVersion     string
		conVersion     string
		expectedResult bool
	}{
		{
			keyVersion:     "1.8.0",
			conVersion:     "1.8.0",
			expectedResult: true,
		},
		{
			keyVersion:     "1.8.1",
			conVersion:     "1.8.0",
			expectedResult: false,
		},
		{
			keyVersion:     "1.7.7",
			conVersion:     "2.0.9",
			expectedResult: false,
		},
	}

	for i, tc := range testCases {
		outResult := IsSemVer(tc.keyVersion, tc.conVersion)
		if tc.expectedResult != outResult {
			t.Fatalf("Failure at test case %d: expected %v, got %v\n", i, tc.expectedResult, outResult)
		}
	}
}
