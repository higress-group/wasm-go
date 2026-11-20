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

package test

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// minWazeroVersion is the oldest wazero release that runs Go 1.24+ wasm
// modules correctly.
//
// wazero <= v1.8.2 corrupts the guest heap when it compiles Go 1.24+ wasm
// modules: on amd64, the wazevo backend clobbers the DX register around
// calls, while Go 1.24+ uses DX in runtime/memmove. The corruption surfaces
// as "runtime: marked free object in span" / "wasm error: unreachable" (or
// as an allocator fatal error) as soon as the guest does enough work, which
// in practice means the wasm-mode unit tests of any plugin using "_rules_"
// crash.
//
// See https://github.com/wazero/wazero/issues/2375 and
// https://github.com/wazero/wazero/pull/2378 (fixed in wazero v1.9.0).
const minWazeroVersion = "v1.9.0"

// TestWazeroVersion guards the wazero dependency floor of this module.
//
// The wasm test host runs plugins inside wazero, so a wazero release that
// miscompiles Go 1.24+ wasm modules breaks the wasm-mode unit tests of this
// repo and of every plugin using it. Behavioural coverage of that failure is
// not reliable - the corruption depends on the exact allocation pattern of
// the guest - so the dependency floor is asserted directly.
//
// To reproduce the crash by hand: build a plugin with
// "GOOS=wasip1 GOARCH=wasm go build -buildmode=c-shared" and configure it
// through the wasm test host with a document containing "_rules_". With
// wazero <= v1.8.2 the guest dies with "wasm error: unreachable".
func TestWazeroVersion(t *testing.T) {
	// When this package is consumed as a dependency, the module that owns the
	// test binary is the plugin's module, not this one: the version that
	// matters is the one the plugin resolves. Only assert the floor for this
	// module itself.
	if inModuleCache(t) {
		t.Skip("this package is used as a dependency; the wazero version of the consuming module is not known here")
	}

	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("the go tool is not available")
	}

	out, err := exec.Command(goBin, "list", "-m", "-f", "{{.Version}}", "github.com/tetratelabs/wazero").Output()
	if err != nil {
		t.Skipf("cannot resolve the wazero version: %v", err)
	}

	version := strings.TrimSpace(string(out))
	require.True(t, versionAtLeast(version, minWazeroVersion),
		"wazero %s miscompiles Go 1.24+ wasm modules (see wazero#2375 and wazero#2378); "+
			"bump github.com/tetratelabs/wazero to at least %s", version, minWazeroVersion)
}

// inModuleCache reports whether this source file lives in the module cache,
// which means the package is consumed as a dependency.
func inModuleCache(t *testing.T) bool {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return false
	}
	return strings.Contains(filepath.ToSlash(file), "/pkg/mod/")
}

// versionAtLeast compares two "vX.Y.Z" versions numerically. A plain string
// comparison would put v1.10.0 before v1.9.0.
func versionAtLeast(version, minimum string) bool {
	got := parseVersion(version)
	want := parseVersion(minimum)
	for i := 0; i < len(want); i++ {
		if got[i] != want[i] {
			return got[i] > want[i]
		}
	}
	return true
}

// parseVersion parses the leading "vX.Y.Z" of a module version.
func parseVersion(version string) [3]int {
	var parsed [3]int
	trimmed := strings.TrimPrefix(version, "v")
	if idx := strings.IndexAny(trimmed, "-+"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	for i, part := range strings.Split(trimmed, ".") {
		if i >= len(parsed) {
			break
		}
		value, err := strconv.Atoi(part)
		if err != nil {
			break
		}
		parsed[i] = value
	}
	return parsed
}
