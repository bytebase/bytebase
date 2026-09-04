#!/bin/bash
# GCE startup-script: root, every boot, idempotent. No `set -e`: a failed step must
# not abandon the boot; `journalctl -t devbox-startup` has the exit status.
set -uo pipefail
trap 'logger -t devbox-startup "exited: status $?"' EXIT
# A fatal exit powers the box off. Left RUNNING with no runners, the start workflow
# would never touch it; off, it is restarted and this script retried on the
# workflow's next tick. Five minutes keeps SSH usable for the journal or /home first.
fatal() {
  logger -t devbox-startup "FATAL: $1"; systemctl stop docker.socket docker containerd 2>/dev/null
  shutdown -h +5 "devbox-startup: $1"; exit 1
}

# Docker down and masked until (a) has mounted /scratch. daemon.json persists, so the
# previous boot's dockerd is already up with its data root on the boot disk, and
# mounting over it would leave it writing there unseen. Masked, nothing can restart it
# -- not apt's postinst, not a connection to a still-listening socket -- until (a) is done.
systemctl mask --now docker.socket docker containerd 2>/dev/null

# ---------- packages ----------
export DEBIAN_FRONTEND=noninteractive
echo 'DPkg::Lock::Timeout "300";' > /etc/apt/apt.conf.d/99-lock-timeout   # unattended-upgrades holds the lock at boot
PKGS="docker.io cron systemd-oomd build-essential mdadm"   # build-essential: Go defaults to CGO_ENABLED=1
dpkg --configure -a 2>/dev/null || true   # clears a transaction a preemption cut short; apt repairs the rest
apt-get -y -qq -f install $PKGS 2>/dev/null || { apt-get update -qq && apt-get -y -qq -f install $PKGS; } \
  || fatal "prerequisite install failed"

# ---------- (a) disk layout ----------
mkdir -p /scratch
DEVS=(); for d in /dev/disk/by-id/google-local-*; do [[ -b $d && $d != *-part* ]] && DEVS+=("$d"); done
DEV=${DEVS[0]:-}
# N2 refuses a single Local SSD at 20 vCPU -- 0, 2, 4, 8, 16 or 24 only -- so there are
# always at least two, and they are striped into one /scratch. A reboot keeps Local SSD
# and udev may have re-assembled the array under a name of its choosing, so look for a
# live array, then for an assemblable one, and only then create: `mdadm --create` over
# an existing array discards it.
if (( ${#DEVS[@]} > 1 )); then
  md() { mdadm --detail --scan 2>/dev/null | awk 'NR==1{print $2}'; }
  DEV=$(md)
  [[ -n "$DEV" ]] || { mdadm --assemble --scan >/dev/null 2>&1; DEV=$(md); }
  [[ -n "$DEV" ]] || { mdadm --create /dev/md0 --level=0 --raid-devices="${#DEVS[@]}" \
                         --run "${DEVS[@]}" >/dev/null 2>&1 && DEV=/dev/md0; }
fi
if [[ -n "$DEV" ]] && ! mountpoint -q /scratch; then
  # A stop discards Local SSD; a reboot does not. Format only when there is no filesystem.
  [[ -n "$(blkid -s TYPE -o value "$DEV")" ]] || mkfs.ext4 -F -m 0 -E lazy_itable_init=0,lazy_journal_init=0,discard "$DEV"
  mount -o noatime,discard "$DEV" /scratch
fi
# Everything below assumes it. Falling back to the boot disk would put swap, Docker,
# work trees and caches on the 100 GB disk that holds /home.
mountpoint -q /scratch || fatal "no Local SSD at /scratch"
chmod 1777 /scratch                                   # OS Login accounts create their own subtree
mkdir -p /scratch/tmp && chmod 1777 /scratch/tmp
# Compare inodes, not `mountpoint`: a tmpfs /tmp is a mountpoint too, and would leave temp files in RAM.
[ "$(stat -c '%d:%i' /tmp)" = "$(stat -c '%d:%i' /scratch/tmp)" ] || mount --bind /scratch/tmp /tmp \
  || fatal "/tmp not on Local SSD"
swapon --show=NAME --noheadings | grep -q /scratch/swapfile \
  || { rm -f /scratch/swapfile && fallocate -l 8G /scratch/swapfile && chmod 600 /scratch/swapfile \
       && mkswap /scratch/swapfile >/dev/null && swapon /scratch/swapfile; } \
  || logger -t devbox-startup "WARN: no swap active"
sysctl -qw vm.swappiness=10
systemctl enable --now systemd-oomd || logger -t devbox-startup "WARN: systemd-oomd inert"
# On the slice: containers get their own scopes under system.slice, not under the runner unit.
mkdir -p /etc/systemd/system/system.slice.d
printf '[Slice]\nManagedOOMMemoryPressure=kill\n' > /etc/systemd/system/system.slice.d/oomd.conf

# Docker on Local SSD with log rotation. The socket is opened to every account: OS
# Login users appear on first connect and cannot be pre-added to the docker group.
mkdir -p /scratch/docker /scratch/containerd /var/lib/containerd /etc/systemd/system/docker.service.d
echo '{"data-root":"/scratch/docker","log-driver":"json-file","log-opts":{"max-size":"10m","max-file":"3"}}' > /etc/docker/daemon.json
printf '[Service]\nExecStartPost=/bin/chmod 666 /var/run/docker.sock\n' > /etc/systemd/system/docker.service.d/socket.conf
# Engine 29 keeps images in containerd's own store, which data-root does not cover. A
# bind needs no containerd config to track.
mountpoint -q /var/lib/containerd || mount --bind /scratch/containerd /var/lib/containerd \
  || fatal "containerd store not on Local SSD"
# Masked until here: an active docker.socket cannot spawn a masked service, so nothing
# can start dockerd on the boot disk between the stop above and this restart. And
# disabled from here on: only this script starts Docker, so a reboot no longer brings
# it up on the boot disk before the script has run.
systemctl unmask docker.socket docker containerd 2>/dev/null
systemctl disable docker.socket docker containerd 2>/dev/null \
  || logger -t devbox-startup "WARN: docker autostart still enabled; the mask covers the next boot"
systemctl daemon-reload && systemctl restart containerd docker || fatal "docker would not start"
docker info --format '{{.DockerRootDir}}' 2>/dev/null | grep -q '^/scratch/' \
  || fatal "docker not on Local SSD"   # daemon.json unwritten or rejected: dockerd starts fine on the boot disk

# ---------- (b) accounts and cache paths ----------
for u in runner1 runner2 runner3 runner4; do
  # Home on scratch: whatever a job writes home-relative is disposable too.
  id "$u" &>/dev/null || useradd -M -d "/scratch/$u/home" -s /usr/sbin/nologin "$u"
  # runner/ up front: systemd applies WorkingDirectory before ExecStartPre could create it.
  mkdir -p "/scratch/$u"/{cache,work,runner,home}
  chown "$u:$u" "/scratch/$u" "/scratch/$u"/{cache,work,runner,home}   # not -R: contents are the user's own
done

# Interactive accounts appear at first OS Login connect, so the profile provisions them:
# each tool's default cache path becomes a symlink onto scratch. Symlinks, not variables,
# so an agent's `bash -c` -- which sources no profile -- lands there too. PATH goes in
# /etc/environment for the same reason: pam_env applies it to every session.
echo 'PATH="/usr/local/go/bin:/usr/local/node/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"' > /etc/environment
cat > /etc/profile.d/devbox.sh <<'EOF'
u=$(id -un); c=/scratch/$u/cache
if mountpoint -q /scratch && [ -n "$u" ] && [ -n "${HOME:-}" ] && mkdir -p "$c" 2>/dev/null; then
  for pair in "$HOME/.cache:$c" "$HOME/go/pkg/mod:$c/go-mod" "$HOME/.npm:$c/npm" "$HOME/.local/share/pnpm:$c/pnpm"; do
    link=${pair%%:*}; target=${pair#*:}
    mkdir -p "$target"   # every login: scratch targets vanish on a stop, the symlink in HOME does not
    [ -L "$link" ] || { mkdir -p "$(dirname "$link")" && rm -rf "$link" && ln -s "$target" "$link"; }
  done 2>/dev/null
fi
unset u c pair link target
EOF
# Existing accounts keep their symlinks across a stop but not the targets. Remake them,
# so only a brand-new account needs a first interactive login.
for h in /home/*; do
  u=${h##*/}; id "$u" &>/dev/null || continue
  for d in "" /go-mod /npm /pnpm; do install -d -o "$u" -g "$u" "/scratch/$u/cache$d"; done
done

# ---------- gh ----------
# Upstream, not apt: Ubuntu 24.04 ships 2.45.0. /usr/local/bin is on the boot disk, so
# this is a no-op on a reboot and only refetches after a rebuild or a new release.
# Never fatal -- gh is a convenience, and a boot must not turn on GitHub being reachable.
GH=$(curl -sf --retry 3 --max-time 30 https://api.github.com/repos/cli/cli/releases/latest \
     | python3 -c 'import sys,json; print(json.load(sys.stdin)["tag_name"].lstrip("v"))' 2>/dev/null) \
  || GH=$(gh --version 2>/dev/null | awk 'NR==1{print $3}')
if [[ -n "${GH:-}" && "$(gh --version 2>/dev/null | awk 'NR==1{print $3}')" != "$GH" ]]; then
  curl -fsSL --retry 3 "https://github.com/cli/cli/releases/download/v$GH/gh_${GH}_linux_amd64.tar.gz" \
    | tar xz -C /tmp \
    && install -m 0755 "/tmp/gh_${GH}_linux_amd64/bin/gh" /usr/local/bin/gh \
    || logger -t devbox-startup "WARN: gh $GH install failed"
  rm -rf "/tmp/gh_${GH}_linux_amd64"
fi

# ---------- (c) GitHub runners ----------
# Stateless, so it lives on scratch and is installed from the unit rather than from
# here: systemd retries that step forever, so a failed download heals itself.
cat > /usr/local/sbin/install-runner <<'EOF'
#!/bin/bash
# <account>. Root, from the unit. Re-runs whole on every retry, deps included.
set -euo pipefail
DIR=/scratch/$1/runner
# Latest release; if the lookup fails, keep what is installed, or fall back to a pin.
V=$(curl -sf --retry 3 --max-time 30 https://api.github.com/repos/actions/runner/releases/latest \
    | python3 -c 'import sys,json; print(json.load(sys.stdin)["tag_name"].lstrip("v"))' 2>/dev/null) \
  || V=$(cat "$DIR/.installed" 2>/dev/null || echo 2.336.0)
# Version-stamped and written last: a reboot keeps scratch, so a bare marker would pin
# the old release, and an unstamped one would skip a failed dep step.
[[ "$(cat "$DIR/.installed" 2>/dev/null)" == "$V" ]] && exit 0
rm -rf "$DIR" && install -d -o "$1" -g "$1" "$DIR"   # clean tree: an old release's files must not linger
curl -fsSL --retry 3 "https://github.com/actions/runner/releases/download/v$V/actions-runner-linux-x64-$V.tar.gz" | tar xz -C "$DIR"
chown -R "$1:$1" "$DIR"
"$DIR"/bin/installdependencies.sh >/dev/null   # apt is a no-op once satisfied
echo "$V" > "$DIR/.installed"
EOF

cat > /usr/local/sbin/register-runner <<'EOF'
#!/bin/bash
# <account>. Runs as the account, every boot.
set -euo pipefail
ORG=bytebase
md() { curl -sf -H "Metadata-Flavor: Google" "http://metadata.google.internal/computeMetadata/v1/$1"; }
PROJECT=$(md project/project-id)
ACCESS=$(md instance/service-accounts/default/token | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])')
PAT=$(curl -sf -H "Authorization: Bearer $ACCESS" \
  "https://secretmanager.googleapis.com/v1/projects/${PROJECT}/secrets/gh-runner-pat/versions/latest:access" \
  | python3 -c 'import sys,base64,json; print(base64.b64decode(json.load(sys.stdin)["payload"]["data"]).decode())')
TOKEN=$(curl -sfX POST -H "Authorization: Bearer $PAT" -H "Accept: application/vnd.github+json" \
  "https://api.github.com/orgs/${ORG}/actions/runners/registration-token" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')
cd "/scratch/$1/runner"
# config.sh refuses to run over an existing config, and --replace only settles a name
# collision on GitHub's side, not this check. Glob both families rather than naming
# files: 2.337.0 migrated its config and left a .runner_migrated marker no name list
# knew to clear, which turned every service restart into a 30s crash loop. A glob
# cannot rot the same way. Nothing else in the tree starts with either prefix.
rm -f .runner* .credentials*
./config.sh --unattended --replace --url "https://github.com/${ORG}" --token "$TOKEN" \
  --name "$(hostname -s)-$1" --work "/scratch/$1/work"   # host-qualified name: never collides with another box
EOF
chmod +x /usr/local/sbin/install-runner /usr/local/sbin/register-runner

cat > /etc/systemd/system/actions-runner@.service <<'EOF'
[Unit]
Description=GitHub Actions runner (%i)
Wants=network-online.target
After=network-online.target docker.service
[Service]
User=%i
WorkingDirectory=/scratch/%i/runner
Environment=XDG_CACHE_HOME=/scratch/%i/cache GOMODCACHE=/scratch/%i/cache/go-mod npm_config_cache=/scratch/%i/cache/npm PNPM_CONFIG_STORE_DIR=/scratch/%i/cache/pnpm
ExecStartPre=+/usr/local/sbin/install-runner %i
ExecStartPre=/usr/local/sbin/register-runner %i
ExecStart=/scratch/%i/runner/run.sh
# install-runner fetches ~200 MB and may wait on the apt lock; the 90s default would
# kill it, and the retry restarts the download from zero.
TimeoutStartSec=600
Restart=always
# Registration failure retries forever; at 5s x3 runners that would burn the API quota.
RestartSec=30
EOF

# ---------- (d) cache cleanup ----------
cat > /usr/local/sbin/cache-gc <<'EOF'
#!/bin/bash
# Every 30 min. Leaked go-build sandboxes are swept always. Above 85% used, wait for an
# idle box -- unless it is nearly full -- then delete everything disposable. It all
# rebuilds on the next job; a full disk does not.
set -uo pipefail
mountpoint -q /scratch || exit 0   # cron outlives a stop; before (a) has mounted, there is nothing to clean
HIGH=85 CRITICAL=95
find /tmp -maxdepth 1 -name 'go-build*' -type d -mmin +1440 -exec rm -rf {} +
used() { df -P /scratch | awk 'NR==2 {print $5+0}'; }
# Load catches detached work -- tmux, nohup, a container -- that holds neither an SSH
# session nor a Runner.Worker. On 20 vCPU, anything actually working exceeds 2.
idle() { ! pgrep -f Runner.Worker >/dev/null && [[ -z "$(ss -Htn state established '( sport = :22 )')" ]] \
         && (( $(cut -d. -f1 /proc/loadavg) < 2 )); }

[[ $(used) -ge $HIGH ]] || exit 0
if ! idle && [[ $(used) -lt $CRITICAL ]]; then logger -t cache-gc "deferred: busy at $(used)%"; exit 0; fi
logger -t cache-gc "cleaning: $(used)% used"
rm -rf /tmp/* /tmp/.[!.]*   # /tmp is on scratch; the box is idle
docker ps -q | xargs -r docker rm -f >/dev/null 2>&1   # idle, so a running container is orphaned
docker system prune -af --volumes >/dev/null 2>&1
# Everything on scratch is disposable except the runner installs, which are large and
# would otherwise be re-downloaded on the next boot.
rm -rf /scratch/*/cache /scratch/*/home /scratch/*/work
for u in runner1 runner2 runner3 runner4; do
  install -d -o "$u" -g "$u" /scratch/$u/{cache,home,work}   # the units need them; a profile remakes an interactive account's
done
logger -t cache-gc "cleaned -> $(used)%"
EOF
chmod +x /usr/local/sbin/cache-gc
# PATH first: cron's default omits /usr/sbin, and a missing `ss` would make idle() look true.
printf 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\n*/30 * * * * root /usr/local/sbin/cache-gc\n' > /etc/cron.d/devbox

systemctl daemon-reload
systemctl restart actions-runner@{runner1,runner2,runner3,runner4}

# Last and time-bounded: observability must not hold up CI capacity.
systemctl is-active --quiet google-cloud-ops-agent \
  || { curl -sS --retry 3 --max-time 120 -o /tmp/ops.sh https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh \
       && timeout 300 bash /tmp/ops.sh --also-install; } \
  || logger -t devbox-startup "WARN: Ops Agent install failed; no memory or swap metrics"
