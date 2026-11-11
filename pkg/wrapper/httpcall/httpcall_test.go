// Copyright (c) 2026 Alibaba Group Holding Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package httpcall

import "testing"

type testCluster struct{}

func (testCluster) ClusterName() string { return "test" }
func (testCluster) HostName() string    { return "test.example.com" }

func TestNewClusterClient(t *testing.T) {
	client := NewClusterClient(testCluster{})
	var _ HttpClient = client
	if got := client.ClusterName(); got != "test" {
		t.Fatalf("ClusterName() = %q, want %q", got, "test")
	}
}
