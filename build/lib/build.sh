#!/usr/bin/env bash

# Tencent is pleased to support the open source community by making TKEStack
# available.
#
# Copyright (C) 2012-2019 Tencent. All Rights Reserved.
#
# Licensed under the Apache License, Version 2.0 (the "License"); you may not use
# this file except in compliance with the License. You may obtain a copy of the
# License at
#
# https://opensource.org/licenses/Apache-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
# WARRANTIES OF ANY KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations under the License.

set -o errexit
set -o nounset
set -o pipefail

PKG=${PKG:-"github.com/yaoice/sample-device-plugin"}
GOBUILD_FLAGS=${debug:+"-v"}
GO_LDFLAGS=${GO_LDFLAGS:-""}

ARCH=${ARCH:-"amd64"}
ROOT_DIR=${ROOT_DIR:-"$(cd $(dirname ${BASH_SOURCE})/../.. && pwd -P)"} 
OUTPUT_DIR=${OUTPUT_DIR:-"${ROOT_DIR}/_output"}
BIN_DIR=${OUTPUT_DIR}/bin-${ARCH}
mkdir -p ${BIN_DIR}

function build::sample-device-plugin() {
  local BIN_PREFIX="sample-device-plugin"

  # build sample-device-plugin
  echo "Building sample-device-plugin"
  echo "   sample-device-plugin"
  echo go build -o ${BIN_DIR}/sample-device-plugin ${GOBUILD_FLAGS} -ldflags "${GO_LDFLAGS}" ${PKG}/cmd/sample-device-plugin
  go build -o ${BIN_DIR}/sample-device-plugin ${GOBUILD_FLAGS} -ldflags "${GO_LDFLAGS}" ${PKG}/cmd/sample-device-plugin
}

function build::verify() {
  bad_files=$(gofmt -s -l pkg/ cmd/)
  if [[ -n "${bad_files}" ]]; then
    echo "gofmt -s -w' needs to be run on the following files: "
    echo "${bad_files}"
    exit 1
  fi
}

(
  for arg; do
    case $arg in
    sample-device-plugin)
      build::sample-device-plugin
      ;;
    verify)
      build::verify
      ;;
    *)
      echo unknown arg $arg
    esac
  done
)
