#!/usr/bin/env bash

### Helpers begin
check_deps() {
    local missing
    for d in "${deps[@]}"; do
        if [[ -z $(command -v "$d") ]]; then
            # Force absolute path
            if [[ ! -e "/$d" ]]; then
                err "$d was not found"
                missing="true"
            fi
        fi
    done; unset d
    [[ -z $missing ]] || exit 128
}
err() { echo -e "${color:+\e[31m}[!] $*\e[0m"; }
errx() { err "${*:2}"; exit "$1"; }
good() { echo -e "${color:+\e[32m}[+] $*\e[0m"; }
info() { echo -e "${color:+\e[37m}[*] $*\e[0m"; }
long_opt() {
    local arg shift="0"
    case "$1" in
        "--"*"="*) arg="${1#*=}"; [[ -n $arg ]] || return 127 ;;
        *) shift="1"; shift; [[ $# -gt 0 ]] || return 127; arg="$1" ;;
    esac
    echo "$arg"
    return $shift
}
subinfo() { echo -e "${color:+\e[36m}[=] $*\e[0m"; }
warn() { echo -e "${color:+\e[33m}[-] $*\e[0m"; }
### Helpers end

usage() {
    cat <<EOF
Usage: ${0##*/} [OPTIONS]

DESCRIPTION
    Simple FTP listener via Docker.

OPTIONS
    -c, --cert=PEM    Use specified TLS cert
    -h, --help        Display this help message
    -k, --key=PEM     Use specified TLS key
        --no-color    Disable colorized output

EOF
    exit "$1"
}

declare -a args
unset help
color="true"

# Parse command line options
while [[ $# -gt 0 ]]; do
    case "$1" in
        "--") shift; args+=("$@"); break ;;
        "-c"|"--cert"*) cert="$(long_opt "$@")" ;;
        "-h"|"--help") help="true" ;;
        "-k"|"--key"*) key="$(long_opt "$@")" ;;
        "--no-color") unset color ;;
        *) args+=("$1") ;;
    esac
    case "$?" in
        0) ;;
        1) shift ;;
        *) usage $? ;;
    esac
    shift
done
[[ ${#args[@]} -eq 0 ]] || set -- "${args[@]}"

# Help info
[[ -z $help ]] || usage 0

# Check for missing dependencies
declare -a deps
deps+=("docker")
check_deps

# Check for valid params
[[ $# -eq 0 ]] || usage 1
[[ -z $cert ]] || [[ -f "$cert" ]] || errx 2 "$cert does not exist"
[[ -z $key ]] || [[ -f "$key" ]] || errx 3 "$key does not exist"

conf="vsftpd.conf"
[[ -z $cert ]] || [[ -z $key ]] || conf="vsftpd_ssl.conf"

mkdir -p ftp
chmod 777 ftp

docker run \
    -e FTP_PASSWORD="ftptest" \
    -e FTP_USER="ftptest" \
    -i --network="host" --rm -t \
    ${cert:+-v "$(pwd)/$cert":/etc/ssl/certs/vsftpd.crt:ro} \
    ${key:+-v "$(pwd)/$key":/etc/ssl/private/vsftpd.key:ro} \
    -v "$(pwd)/ftp":/srv \
    panubo/vsftpd vsftpd "/etc/$conf"
