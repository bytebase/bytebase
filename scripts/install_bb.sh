#! /bin/sh
set -u

abort() {
    printf "%s\n" "$@" >&2
    exit 1
}

uname_os() {
    OS="$(uname -s)"
    if [[ "${OS}" != "Darwin" && "${OS}" != "Linux" ]]; then
        abort "OS ${OS} is not support, bb is only supported on Linux and MacOS"
    fi
    echo $OS
}

uname_arch() {
    ARCH=$(uname -m)
    if [[ "${ARCH}" == "amd64" || "${ARCH}" == "x86_64" ]]; then
        ARCH="x86_64"
    elif [[ "${ARCH}" != "arm64" ]]; then
        abort "Arch ${ARCH} is not support, bb is only supported on x86_64, amd64 and arm64"
    fi
    echo $ARCH
}

http_download() {
    local_file=$1
    source_url=$2

    echo "Start downloading $source_url..."

    code=$(curl -w '%{http_code}' -sL -o "$local_file" "$source_url")
    if [ "$code" != "200" ]; then
        abort "Failed to download $source_url, status code: $code"
    fi

    echo "Completed downloading $source_url"
    return 0
}

execute() {
    BB_VERSION="1.1.0"
    OS="$(uname_os)"
    ARCH="$(uname_arch)"

    tmpdir=$(mktemp -d)
    echo "Downloading tarball into ${tmpdir}"

    http_download "${tmpdir}/bytebase_${BB_VERSION}_${OS}_${ARCH}.tar.gz" \
        "https://github.com/bytebase/bytebase/releases/download/${BB_VERSION}/bytebase_${BB_VERSION}_${OS}_${ARCH}.tar.gz"

    echo "Start extracting tarball..."
    (cd "${tmpdir}" && tar -xf "bytebase_${BB_VERSION}_${OS}_${ARCH}.tar.gz" "bb")
    echo "Completed extracting tarball"

    echo "Start installing bb..."
    echo "Requesting root privilege..." 
    sudo install "${tmpdir}/bb" "/usr/local/bin"
    echo "Installed bb to /usr/local/bin"

    echo ""
    echo "Check the usage with"
    echo "  bb --help"
    echo ""
    echo "You can clean up the installation files with"
    echo "  rm -rf ${tmpdir}"
}

execute
