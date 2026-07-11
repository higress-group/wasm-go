#!/usr/bin/env bash

set -euo pipefail

usage() {
  cat <<'EOF'
Usage: build-wasm.sh [-o OUTPUT] [PACKAGE]

Build a Go package for wasip1/wasm and optimize the release artifact with
Binaryen wasm-opt -Oz. PACKAGE defaults to the current directory and OUTPUT
defaults to main.wasm.

Environment variables:
  GO_BIN             Go executable to use (default: go)
  WASM_OPT           wasm-opt executable to use (default: wasm-opt)
  BINARYEN_VERSION   required Binaryen version (default: 130)
  GO_BUILD_TAGS      optional Go build tags, for example wasm_lite_http
EOF
}

output=main.wasm
package=.
package_set=false

while (($# > 0)); do
  case "$1" in
    -o|--output)
      if (($# < 2)); then
        echo "missing value for $1" >&2
        usage >&2
        exit 2
      fi
      output=$2
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      if (($# > 1)); then
        echo "only one package may be specified" >&2
        usage >&2
        exit 2
      fi
      if (($# == 1)); then
        package=$1
      fi
      break
      ;;
    -*)
      echo "unknown option: $1" >&2
      usage >&2
      exit 2
      ;;
    *)
      if [[ ${package_set} == true ]]; then
        echo "only one package may be specified" >&2
        usage >&2
        exit 2
      fi
      package=$1
      package_set=true
      shift
      ;;
  esac
done

go_bin=${GO_BIN:-go}
wasm_opt=${WASM_OPT:-wasm-opt}
binaryen_version=${BINARYEN_VERSION:-130}
go_build_tags=${GO_BUILD_TAGS:-}

if ! command -v "${go_bin}" >/dev/null 2>&1; then
  echo "Go executable not found: ${go_bin}" >&2
  exit 1
fi
if ! command -v "${wasm_opt}" >/dev/null 2>&1; then
  echo "wasm-opt not found: ${wasm_opt}; install Binaryen ${binaryen_version} first" >&2
  exit 1
fi

wasm_opt_version=$("${wasm_opt}" --version 2>&1) || {
  echo "failed to execute ${wasm_opt} --version" >&2
  exit 1
}
expected_version_prefix="wasm-opt version ${binaryen_version} "
if [[ ${wasm_opt_version} != "${expected_version_prefix}"* ]]; then
  echo "expected ${expected_version_prefix}but got: ${wasm_opt_version}" >&2
  exit 1
fi

output_dir=$(dirname -- "${output}")
output_name=$(basename -- "${output}")
mkdir -p "${output_dir}"
raw_wasm=$(mktemp "${output_dir}/.${output_name}.unoptimized.XXXXXX")
optimized_wasm=$(mktemp "${output_dir}/.${output_name}.optimized.XXXXXX")

cleanup() {
  rm -f "${raw_wasm}" "${optimized_wasm}"
}
trap cleanup EXIT

go_build_args=(
  -trimpath
  -buildmode=c-shared
  -ldflags='-s -w -buildid='
)
if [[ -n ${go_build_tags} ]]; then
  go_build_args+=(-tags "${go_build_tags}")
fi
go_build_args+=(-o "${raw_wasm}" "${package}")

GOOS=wasip1 GOARCH=wasm "${go_bin}" build "${go_build_args[@]}"

"${wasm_opt}" "${raw_wasm}" \
  -Oz \
  --enable-bulk-memory \
  -o "${optimized_wasm}"

chmod 0644 "${optimized_wasm}"
mv -f "${optimized_wasm}" "${output}"
echo "Built optimized Wasm artifact: ${output}"
