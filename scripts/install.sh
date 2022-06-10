#!/bin/sh

set -u

abort() {
    printf "%s\n" "$@" >&2
    exit 1
}

uname_os() {
    local OS="$(uname -s)"
    if [[ "${OS}" != "Darwin" && "${OS}" ]]; then
        abort "OS ${OS} is not support, bytebase is only supported on Linux and MacOS"
    fi
    echo ${OS}
}

uname_arch() {
    local ARCH=$(uname -m)
    if [[ "${ARCH}" == "amd64" || "${ARCH}" == "x86_64" ]]; then
        ARCH="x86_64"
    elif [[ "${ARCH}" != "arm64" ]]; then
        abort "Arch ${ARCH} is not support, bytebase is only supported on x86_64, amd64 and arm64"
    fi
    echo ${ARCH}
}

test_curl() {
    local curl_version=$(curl --version 2>/dev/null)
    if [ $? -ne 0 ]; then
        abort "You must install curl before installing bytebase."
    fi
}

test_tar() {
    local tar_version=$(tar --version 2>/dev/null)
    if [ $? -ne 0 ]; then
        abort "You must install tar before installing bytebase."
    fi
}

http_download() {
    local local_file=$1
    local source_url=$2

    echo "Start downloading ${source_url}..."

    local code=$(curl -w '%{http_code}' -L -o "${local_file}" "${source_url}")
    if [ "$code" != "200" ]; then
        abort "Failed to download from ${source_url}, status code: ${code}"
    fi

    echo "Completed downloading ${source_url}"
}

get_bytebase_latest_version() {
    local version_url="https://raw.githubusercontent.com/bytebase/bytebase/main/scripts/VERSION"
    local local_file=$1

    local code=$(curl -w '%{http_code}' -sL -o "${local_file}" "${version_url}")
    if [ "$code" != "200" ]; then
        abort "Failed to get bytebase latest version from ${version_url}, status code: ${code}"
    fi

    local version=$(<${local_file})

    echo "${version}"
}

execute() {
    OS="$(uname_os)"
    echo "OS: ${OS}"
    ARCH="$(uname_arch)"
    echo "ARCH: ${ARCH}"

    test_curl
    test_tar

    # Initialize bytebase direcoty
    bytebase_dir="/opt/bytebase"
    (sudo mkdir -p "${bytebase_dir}") || abort "cannot create directory ${bytebase_dir}"

    install_dir="/usr/local/bin"

    tmp_dir=$(mktemp -d) || abort "cannot create temp directory"
    # Clean the tmpdir automatically if the shell script exit
    trap "rm -r ${tmp_dir}" EXIT

    BYTEBASE_VERSION="$(get_bytebase_version ${tmp_dir}/VERSION)"
    echo "Get bytebase latest version: ${BYTEBASE_VERSION}"

    echo "Downloading tarball into ${tmp_dir}"
    tarball_name="bytebase_${BYTEBASE_VERSION}_${OS}_${ARCH}.tar.gz"
    http_download "${tmp_dir}/${tarball_name}" \
        "https://github.com/bytebase/bytebase/releases/download/${BYTEBASE_VERSION}/${tarball_name}"

    echo "Start extracting tarball into ${bytebase_dir}..."
    cd "${bytebase_dir}" && sudo tar -xzf "${tmp_dir}/${tarball_name}"

    echo "Start installing bytebase and bb ${VERSION}"
    sudo install "${bytebase_dir}/bytebase" "${install_dir}"
    echo "Installed bytebase ${VERSION} to ${install_dir}"
    sudo install "${bytebase_dir}/bb" "${install_dir}"
    echo "Installed bb ${VERSION} to ${install_dir}"
    echo ""
    echo "Check the usage with"
    echo "  bytebase --help"
    echo "  bb --help"
}

execute
