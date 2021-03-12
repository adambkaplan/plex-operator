#!/usr/bin/env bash

# Copyright Adam B Kaplan
# 
# SPDX-License-Identifier: Apache-2.0

set -e

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..

function listPkgDirs() {
	go list -f '{{.Dir}}' ./api/... ./controllers/... ./pkg/...
  local goFiles=$?
}

function listGoFiles() {
	# pipeline is much faster than for loop
	listPkgDirs | xargs -I {} find {} \( -name '*.go' -a ! -name "zz_generated*.go" \)
  local goFiles=$?
  echo "${SCRIPT_ROOT}/main.go"
  goFiles="$goFiles $?"
}

function generateGoCopyright() {
  allFiles=$(listGoFiles)

  for file in $allFiles ; do
    if ! head -n3 "${file}" | grep -Eq "(Copyright|SPDX-License-Identifier)" ; then
      cp "${file}" "${file}.bak"
      cat "${SCRIPT_ROOT}/hack/boilerplate.go.txt" > "${file}"
      cat "${file}.bak" >> "${file}"
      rm "${file}.bak"
    fi
  done
}

generateGoCopyright

set +e
