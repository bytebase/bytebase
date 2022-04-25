#! /bin/sh
set -u

PREV_WD="$(pwd)"
mkdir -p .bytebase && cd .bytebase
abort() {
    cd $PREV_WD && rm -rf .bytebase
    printf "%s\n" "$@" >&2
    exit 1
}

BB_VERSION="1.0.3"

OS="$(uname -s)"
if [[ "${OS}" != "Darwin" && "${OS}" != "Linux" ]]; then
    abort "OS ${OS} is not support, bb is only supported on Linux and MacOS"
fi

ARCH=$(uname -m)
if [[ "${ARCH}" == "amd64" || "${ARCH}" == "x86_64" ]]; then
    ARCH="x86_64"
elif [[ "${ARCH}" != "arm64" ]]; then
    abort "Arch ${ARCH} is not support, bb is only supported on x86_64, amd64 and arm64"
fi

download_tar() {
    curl -LO "https://github.com/bytebase/bytebase/releases/download/${BB_VERSION}/bytebase_${BB_VERSION}_${OS}_${ARCH}.tar.gz"
}

extract_tar() {
    tar -xf "bytebase_${BB_VERSION}_${OS}_${ARCH}.tar.gz" "bb"
}

install_bb_binary() {
    sudo mv "bb" "/usr/local/bin/bb"
    echo "bb ${BB_VERSION} has been installed in /usr/local/bin"
}

download_tar
extract_tar
install_bb_binary
cd $PREV_WD && rm -rf .bytebase
bb version &>/dev/null
exit $@
