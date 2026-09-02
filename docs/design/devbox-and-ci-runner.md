# Combined Dev Box and CI Runner on GCP

One Spot VM with two jobs. It is the SSH target that agentic coding tools drive, and
it hosts three self-hosted GitHub Actions runners for the whole org.

## Goals

1. **The cheapest VM that still performs.** Spot, N2 or N2D, 20 vCPU. Sized for
   parallel Go builds and Docker-backed tests. Re-priced per region and family before
   any rebuild.
2. **One startup script, so the box is close to stateless.** Everything on the box is
   produced by that script. Only `/home` and a shared toolchain in `/usr/local` are
   yours to manage; nothing else is set up by hand.
3. **GitHub Actions runners that set themselves up.** Three org-level slots. Each
   re-registers at boot from a secret, so there is no registration state to preserve.
4. **A clear split between Local SSD and the persistent disk.** Local SSD holds what
   can be discarded: caches, Docker, temp files, swap. The boot disk holds repos and
   uncommitted work. It can be lost too, so push your branches.
5. **Cache cleanup that will not break a running job.** Leaked temp files are swept
   always. Caches are evicted only under disk pressure, cheapest to rebuild first.
   The one exception is a disk about to fill, which would fail every job anyway.
6. **Guest memory and swap visible in Cloud Monitoring.** The hypervisor reports only
   CPU and disk. The Ops Agent reports the rest, so RAM is sized from measurement
   instead of a guess.
7. **Usable as a desktop agent's SSH environment.** One SSH config entry is enough.
   A reserved address keeps that entry valid across preemptions. OS Login manages
   keys through IAM, with no bastion. An account that first appears at connect time
   can run Docker and has `sudo`.

## 1. Machine spec and cost

| Property | Value |
|---|---|
| Project / zone | `bytebase-dev` / `northamerica-northeast2-a` (Toronto) |
| Machine type | `n2-custom-20-32768` — 20 vCPU, 32 GB |
| Provisioning | `SPOT`, `instanceTerminationAction=STOP` |
| Image | `ubuntu-2404-lts-amd64` |
| Boot disk | 100 GB pd-balanced, `--no-boot-disk-auto-delete` |
| Scratch | 375 GB Local SSD (NVMe) |
| External IP | reserved; carries all egress, and inbound SSH |
| Inbound | SSH only, via the VPC default rule; no tag, no custom rules |
| Service account | dedicated; reads one secret, writes metrics and logs, nothing else |
| Availability | always on; preemption stops it, the start workflow restarts it within its polling interval |

| Component | Rate | $/mo |
|---|---|---|
| 20 vCPU (N2) | $0.003290 / vCPU-hr | $48.03 |
| 32 GiB RAM (N2) | $0.000441 / GiB-hr | $10.30 |
| 375 GB Local SSD | $0.052800 / GB-mo | $19.80 |
| 100 GB pd-balanced | $0.110 / GB-mo | $11.00 |
| Reserved external IP | $0.0025 / hr | $1.82 |
| **Total** | | **$90.96** |

Catalog prices, 2026-09-01. vCPU and RAM are changeable on a stopped instance with
`set-machine-type`; Local SSD capacity is fixed at creation.

### Re-checking the region

Spot prices are set per family *per region*, and they drift. Re-check before any
rebuild.

Price the whole VM, not just the CPU. Keep only regions that can actually run the
family: the catalog prices N1 in 14 regions that cannot run it. Prefer N2 or N2D,
whichever is cheaper, and use N1 only when both are far more expensive.

Moving is delete and recreate. Push any work in `/home` first; the startup script
rebuilds the rest.

### Create

**1. GitHub PAT.** Fine-grained tokens cannot be created from an API, so this part is
manual — at `github.com/settings/personal-access-tokens/new`:

| Field | Value |
|---|---|
| Resource owner | `bytebase` (the org, not your account) |
| Expiration | set one; rotation should be forced, not remembered |
| Repository access | Public repositories (read-only) |
| Organization permissions → **Self-hosted runners** | **Read and write** |
| Repository permissions | none |

Org-owned tokens may need an org admin to approve the request.

Developer access is granted below rather than by the script, because an OS Login
account does not exist until its owner first connects.

**2. Everything else.**

```bash
PROJECT=bytebase-dev
ZONE=northamerica-northeast2-a
SA=devbox@${PROJECT}.iam.gserviceaccount.com

gcloud services enable secretmanager.googleapis.com oslogin.googleapis.com \
  monitoring.googleapis.com logging.googleapis.com --project=$PROJECT

# Service account; its roles are granted below, nothing more.
gcloud iam service-accounts create devbox --project=$PROJECT \
  --display-name="devbox: dev host and CI runners"

# The PAT, readable by that service account and nothing else.
read -rsp 'GitHub PAT: ' GH_PAT && echo
printf '%s' "$GH_PAT" | gcloud secrets create gh-runner-pat \
  --project=$PROJECT --replication-policy=automatic --data-file=-
unset GH_PAT

gcloud secrets add-iam-policy-binding gh-runner-pat --project=$PROJECT \
  --member="serviceAccount:${SA}" --role=roles/secretmanager.secretAccessor

# Metrics and logs from the Ops Agent. Without these the agent gets 403s and goal 6
# is silently unmet -- a scope is not a role.
for r in monitoring.metricWriter logging.logWriter; do
  gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:${SA}" --role=roles/$r
done

# A reserved address. An ephemeral one is released when the instance stops, so every
# preemption would hand the box a new IP and strand the generated SSH entry.
gcloud compute addresses create devbox --project=$PROJECT --region=${ZONE%-*}

# Each developer who will SSH in. OS Login checks both: osAdminLogin gives the
# account `sudo`, and because the VM has a service account attached, connecting also
# requires permission to act as it -- without that, OS Login refuses the connection
# before it ever creates the account.
for DEV in you@bytebase.com; do
  gcloud projects add-iam-policy-binding $PROJECT \
    --member="user:$DEV" --role=roles/compute.osAdminLogin
  gcloud iam service-accounts add-iam-policy-binding $SA --project=$PROJECT \
    --member="user:$DEV" --role=roles/iam.serviceAccountUser
done

# The instance.
gcloud compute instances create devbox --project=$PROJECT --zone=$ZONE \
  --address=devbox \
  --machine-type=n2-custom-20-32768 \
  --provisioning-model=SPOT --instance-termination-action=STOP \
  --image-family=ubuntu-2404-lts-amd64 --image-project=ubuntu-os-cloud \
  --boot-disk-size=100GB --boot-disk-type=pd-balanced --no-boot-disk-auto-delete \
  --local-ssd=interface=NVME \
  --service-account=$SA --scopes=cloud-platform \
  --metadata=enable-oslogin=TRUE \
  --metadata-from-file=startup-script=./startup.sh
```

The `cloud-platform` scope resolves to the intersection with the account's roles: one
secret, metrics, logs. The scope is not the boundary. The IAM bindings are.

There is no flag day. Runners register at the organization under the automatic
`self-hosted` label, with host-qualified names. So this box only adds capacity
alongside any existing runner, and you can delete the old one whenever.

One step does have to happen in order. A preemption stops the instance, and
`.github/workflows/start-runner-vm.yml` already restarts a runner box on a schedule.
Repoint its instance name and zone at this one, in two places each, as the last step
of cutover. Do it earlier and it stops restarting the box it serves today. Until then,
start this one by hand.

Raise its cadence at the same time. It runs at three fixed hours, so a preemption just
after one of them leaves the box stopped for up to fifteen -- which is not "always on"
for either the dev box or CI. Every fifteen minutes bounds the outage at fifteen
minutes, and starting a running instance is a no-op.

### Connect

Run `gcloud compute config-ssh` once. It writes the `~/.ssh/config` entries, and this
box's alias is `devbox.northamerica-northeast2-a.bytebase-dev`. Desktop agents take
that alias as their SSH host: Claude Desktop under **Add SSH connection**, Codex in
its SSH environment. Each developer gets their own account and home, created by OS
Login on first connect.

Log in once interactively before pointing an agent at a new account. The cache links
in (b) are made by the profile, and a non-login `bash -c` does not read one -- so an
account whose very first session is an agent command would write that session's caches
to the boot disk. After that first login the links exist and every later session
follows them, non-login included -- and the startup script remakes the targets on
every boot, so a preemption does not repeat the exercise. `PATH` needs no such step
at all: it comes from `/etc/environment`, which every session reads.

## 2. Startup script

Saved as `startup.sh` and passed to the create command above. It runs as root on every
boot and is idempotent. Local SSD is discarded on every stop, so anything living there
is rebuilt. Anything on the boot disk is guarded, so it is not redone.

### a. Disk layout

Two kinds of account use the box, and they need opposite things from it.

**Runner accounts** are `runner1`, `runner2` and `runner3`. The script creates them, `nologin`,
one per job slot. Nothing of theirs is worth keeping: software, work trees and caches
all sit on Local SSD, and their home directories are on it too. They are
disposable, and are disposed of on every stop.

**Interactive accounts** arrive through OS Login on first connect. Their `/home/<u>`
is the only durable state on the box: repos, uncommitted changes, agent CLI state,
`~/.ssh`. Nothing backs it up and nothing collects it. Push your branches, and keep an
eye on the boot disk.

| | Runner account | Interactive account |
|---|---|---|
| Created by | the startup script | OS Login, on first connect |
| Boot disk | an unused `/home/<u>` | `/home/<u>` — **the only durable state** |
| Local SSD | `/scratch/<u>/runner`, `/work`, `/cache` | `/scratch/<u>/cache` |
| Cache wiring | `Environment=` in the systemd unit | `/etc/profile.d` symlinks the default paths onto scratch |
| Lost on a rebuild | nothing | uncommitted work |

Local SSD also carries Docker's `data-root`, the swapfile and `/tmp`. Container
startup and database `fsync` are the heaviest I/O on this box. Docker starts before
the script runs, so the script points it at Local SSD and restarts it.

**Swap causes hangs; it does not prevent them.** Without swap, exhausting RAM is a
fast OOM kill. With swap, the box thrashes until the job timeout. So there is 8 GB at
`vm.swappiness=10`: enough to evict cold pages, too little to sustain a thrash. And
`systemd-oomd` kills a runaway job before either happens.

The `monitoring.metricWriter` role puts both in Cloud Monitoring, as
`memory/percent_used` and `swap/percent_used`. Sustained swap above zero means 32 GB
is too little.

Go and Node are not installed here. An agent session installs its own with `sudo`, to
`/usr/local/go` and `/usr/local/node`, which `/etc/environment` puts on the `PATH` of
every session, non-login included. CI jobs bring their own through `setup-go` and
`setup-node`. The C toolchain is the exception and is a prerequisite: neither action
installs one, and Go defaults to `CGO_ENABLED=1`, so dependencies carrying cgo files
fail to build without it.

### b. Cache paths

Four paths carry every cache. `XDG_CACHE_HOME` relocates the Go **build** cache,
Playwright, yarn and pip. Three others ignore it: the Go **module** cache sits under
`GOPATH`, npm uses `~/.npm`, and pnpm's store is under `XDG_DATA_HOME`. `TMPDIR` is
unset, because `/tmp` is bind-mounted onto Local SSD and catches every process.

The paths are per-account. Go writes module-cache files read-only, so one shared
`GOMODCACHE` becomes a permissions problem.

**Runner accounts** get them from `Environment=` in the unit. systemd owns a service's
environment, so the setting always applies. A GitHub Actions `run:` step is a
non-login shell and would never read a profile.

**Interactive accounts** cannot be named in a unit. Instead `/etc/profile.d/devbox.sh`
runs on every login and replaces each tool's default cache location with a symlink
onto scratch:

```
~/.cache             ->  /scratch/<u>/cache
~/go/pkg/mod         ->  /scratch/<u>/cache/go-mod
~/.npm               ->  /scratch/<u>/cache/npm
~/.local/share/pnpm  ->  /scratch/<u>/cache/pnpm
```

It creates the target directory first, then the link, and skips anything that is
already a link. Nothing configures Go, npm or pnpm. They write to the paths they
always use, and the filesystem sends those writes to Local SSD.

Symlinks rather than variables. A variable's reach depends on how the SSH client
invokes the shell. Some tools ignore `XDG_CACHE_HOME` anyway. And a missed variable
fails *silently* onto the boot disk, while a dangling symlink fails loudly. The
targets are recreated on every login, because a symlink on the durable disk pointing
into the disposable one must be re-pointed after each wipe.

Nothing outside the box needs to know any of this. `/tmp` and the default cache paths
already resolve to Local SSD, so `AGENTS.md` stays environment-agnostic. Only output
written *into* the repo tree follows the repo onto the boot disk — `./bytebase-build/`
and `node_modules` — which is right for anything inside a working copy.

### c. GitHub runners

Three accounts, three job slots, one systemd template. A pull request touching both
backend and frontend schedules five self-hosted jobs -- one each from `backend-tests`,
`golangci-lint` and `test_link`, two from `frontend-tests` -- so two of them queue,
before counting any other repository in the org. That is deliberate: jobs queue, they
do not fail, and 20 vCPU split five ways is thin. A fourth or fifth slot is one more
name in the script's account list if the wait proves worse than the contention.

The runner is stateless, so it lives on scratch and is installed by the unit rather
than by the startup script: an `ExecStartPre` runs `install-runner` as root, before
the account registers. systemd retries that step, so a failed download heals itself
instead of leaving the slot dead until someone reboots the box. Each account fetches
its own ~200 MB copy, which costs seconds. In return the boot disk holds nothing of
the runner's, and every start runs the version resolved at boot.

`register-runner` reads the PAT from Secret Manager over the REST API, since the
Ubuntu images ship no `gcloud`. It exchanges the PAT for a registration token and
configures the runner. It deletes `.runner` and `.credentials` first, because
`config.sh` refuses to run over an existing config. `--replace` then takes the
server-side entry of the same name. Names are host-qualified, so two boxes never claim
each other's.

Re-registering rather than preserving credentials means the runner holds no identity
on disk. Wipe the boot disk and the same service account brings back the same three
runners. Registration is derived state, like `/scratch`.

The units carry no `[Install]` section. The script starts them rather than enabling
them, so they cannot race ahead of the scratch mount.

### d. Cache cleanup

Two problems, handled differently. **Garbage** is leaked `go-build*` sandboxes from
cancelled or OOM-killed builds, and it is swept on every run. **Caches** are evicted
only under disk pressure, because they are what make builds fast.

The Go build cache dominates. On a comparable box it was 59% of all disk used. Go
evicts by age with no size cap, so five days of CI output is about 55 GB per account,
and it regrows after any cleanup.

Eviction fires at 85% and deletes the lot: every cache, every home and work tree, and
everything Docker holds. No tiers and no filters. Deciding what to spare would mean
ranking rebuild costs against each other, and re-pulling an image or a module cache
costs minutes where a full disk fails every job on the box.

It waits for an idle box, except above 95% where a failing disk is worse than a failed
job. The idle check is a sample, not a lock, so a job starting mid-sweep can still lose
what it was using. Accepted on the same grounds.

It runs on a schedule, not at boot. A boot-time check cannot fire on a box that stays
up for days, and Local SSD survives a guest *reboot*. It is discarded only on stop or
preemption.

### startup.sh

```bash
#!/bin/bash
# GCE startup-script: root, every boot, idempotent. No `set -e` -- one failed step
# must not abandon the boot; `journalctl -t devbox-startup` shows the exit status.
set -uo pipefail
trap 'logger -t devbox-startup "exited: status $?"' EXIT
# daemon.json persists, so a previous boot's dockerd is already up with
# data-root=/scratch/docker -- on the boot disk until (a) mounts over it. Mounting
# over a live data root is what this prevents, so a stop that did not take is fatal.
# The socket counts: left active it reactivates dockerd on the next connection.
stop_docker() {
  systemctl stop docker.socket docker containerd 2>/dev/null   # absent on a first boot
  if pgrep -x dockerd >/dev/null || pgrep -x containerd >/dev/null \
     || systemctl is-active --quiet docker.socket; then
    logger -t devbox-startup "FATAL: docker would not stop"; exit 1
  fi
}
stop_docker   # before anything that can block or exit
systemctl mask docker.socket docker containerd 2>/dev/null   # postinst must not start them
# Latest runner release, falling back to a known-good pin if the lookup fails. Every
# boot uses this: the runner lives on scratch and is re-fetched, never updated in place.
RUNNER_VERSION=$(curl -sf --retry 3 --max-time 30 https://api.github.com/repos/actions/runner/releases/latest | python3 -c 'import sys,json; print(json.load(sys.stdin)["tag_name"].lstrip("v"))' 2>/dev/null || echo 2.336.0)

# ---------- packages: the Ubuntu image ships neither ----------
export DEBIAN_FRONTEND=noninteractive
echo 'DPkg::Lock::Timeout "300";' > /etc/apt/apt.conf.d/99-lock-timeout   # unattended-upgrades holds the lock at boot
PKGS="docker.io cron systemd-oomd build-essential"
# Unconditional: a preemption during any dpkg work -- an unattended upgrade of some
# unrelated package -- blocks every later apt call, including installdependencies.sh.
# DPkg::Lock::Timeout is apt configuration and dpkg does not honour it, so wait here
# instead: unattended-upgrades holds the lock at boot, which is not a wedged dpkg.
for _ in $(seq 60); do dpkg --configure -a 2>/dev/null && { DPKG_OK=1; break; }; sleep 5; done
[[ -n "${DPKG_OK:-}" ]] || { logger -t devbox-startup "FATAL: dpkg wedged; apt is unusable"; exit 1; }
# Count 'install ok installed', not dpkg -s: a preemption mid-apt leaves packages
# unpacked but unconfigured, which dpkg -s still reports as success.
[[ $(dpkg-query -W -f='${Status}\n' $PKGS 2>/dev/null | grep -c '^install ok installed$') -eq $(wc -w <<<"$PKGS") ]] \
  || { apt-get update -qq && apt-get install -y -qq $PKGS; } \
  || { logger -t devbox-startup "FATAL: prerequisite install failed"; exit 1; }
systemctl unmask docker.socket docker containerd 2>/dev/null
stop_docker   # belt and braces once unmasked

# ---------- (a) disk layout ----------
mkdir -p /scratch
DEV=$(ls /dev/disk/by-id/google-local-* 2>/dev/null | head -1)
if [[ -n "$DEV" ]] && ! mountpoint -q /scratch; then
  # A stop discards Local SSD; a reboot does not. Format only when there is no filesystem.
  [[ -n "$(blkid -s TYPE -o value "$DEV")" ]] || mkfs.ext4 -F -m 0 -E lazy_itable_init=0,lazy_journal_init=0,discard "$DEV"
  mount -o noatime,discard "$DEV" /scratch
fi
# Everything below assumes /scratch is the Local SSD. Falling back to the boot disk
# would put swap, Docker, work trees and caches on the 100 GB disk that holds /home.
mountpoint -q /scratch || { logger -t devbox-startup "FATAL: no Local SSD at /scratch"; exit 1; }
chmod 1777 /scratch                                   # OS Login accounts create their own subtree
mkdir -p /scratch/tmp && chmod 1777 /scratch/tmp
# Bind unless /tmp already *is* /scratch/tmp: `mountpoint` would also be true for a
# tmpfs /tmp, which would silently leave temp files in RAM.
[ "$(stat -c '%d:%i' /tmp)" = "$(stat -c '%d:%i' /scratch/tmp)" ] \
  || mount --bind /scratch/tmp /tmp \
  || { logger -t devbox-startup "FATAL: /tmp not on Local SSD"; exit 1; }
# blkid, not -f: a partly written file satisfies -f and swapon then rejects it.
if [[ "$(blkid -s TYPE -o value /scratch/swapfile 2>/dev/null)" != swap ]]; then
  rm -f /scratch/swapfile
  fallocate -l 8G /scratch/swapfile && chmod 600 /scratch/swapfile && mkswap /scratch/swapfile >/dev/null
fi
swapon /scratch/swapfile 2>/dev/null                  # fails harmlessly if already active
swapon --show=NAME --noheadings | grep -q . || logger -t devbox-startup "WARN: no swap active"
sysctl -qw vm.swappiness=10
systemctl enable --now systemd-oomd || logger -t devbox-startup "WARN: systemd-oomd inert"
# On the slice, not the runner unit: Docker gives each container its own scope under
# system.slice, so a unit-scoped policy would never see the containers under test.
mkdir -p /etc/systemd/system/system.slice.d
printf '[Slice]\nManagedOOMMemoryPressure=kill\n' > /etc/systemd/system/system.slice.d/oomd.conf

# Docker on Local SSD with log rotation. The socket is opened to every account: OS
# Login users appear on first connect and cannot be pre-added to the docker group.
mkdir -p /scratch/docker /etc/systemd/system/docker.service.d
echo '{"data-root":"/scratch/docker","log-driver":"json-file","log-opts":{"max-size":"10m","max-file":"3"}}' > /etc/docker/daemon.json
printf '[Service]\nExecStartPost=/bin/chmod 666 /var/run/docker.sock\n' > /etc/systemd/system/docker.service.d/socket.conf
# Engine 29 defaults to the containerd image store, and data-root does not cover it:
# images and snapshots live under containerd's own root. Bind it rather than edit
# containerd's TOML, so there is no config format or default to track.
mkdir -p /scratch/containerd /var/lib/containerd
mountpoint -q /var/lib/containerd || mount --bind /scratch/containerd /var/lib/containerd \
  || { logger -t devbox-startup "FATAL: containerd store not on Local SSD"; exit 1; }
systemctl daemon-reload && systemctl restart containerd docker

# ---------- (b) accounts and cache paths ----------
for u in runner1 runner2 runner3; do
  # Home on scratch, not /home: whatever a job writes home-relative that the cache
  # variables do not cover is disposable too, and the GC only looks at /scratch.
  id "$u" &>/dev/null || useradd -M -d "/scratch/$u/home" -s /usr/sbin/nologin "$u"
  usermod -aG docker "$u"
  # runner/ too: systemd applies WorkingDirectory before ExecStartPre runs, so the
  # unit would fail on CHDIR before install-runner could create it.
  mkdir -p "/scratch/$u"/{cache,work,runner,home}
  chown "$u:$u" "/scratch/$u" "/scratch/$u"/{cache,work,runner,home}   # not -R: contents are the user's own
done

# Interactive accounts appear on first OS Login connect, so the profile provisions them:
# each tool's default cache path is symlinked onto scratch. Symlinks, not variables, so
# an agent's `bash -c` -- which never sources a profile -- lands there too.
# PATH for every SSH session. pam_env reads this file, so an agent's non-login
# `bash -c` -- which reads no profile -- still finds the shared toolchain. Unlike the
# cache symlinks below, a PATH set in a profile would not outlive the login shell.
echo 'PATH="/usr/local/go/bin:/usr/local/node/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"' > /etc/environment

cat > /etc/profile.d/devbox.sh <<'EOF'
u=$(id -un); c=/scratch/$u/cache
if mountpoint -q /scratch && [ -n "$u" ] && [ -n "${HOME:-}" ] && mkdir -p "$c" 2>/dev/null; then
  for pair in "$HOME/.cache:$c" "$HOME/go/pkg/mod:$c/go-mod" "$HOME/.npm:$c/npm" "$HOME/.local/share/pnpm:$c/pnpm"; do
    link=${pair%%:*}; target=${pair#*:}
    mkdir -p "$target"   # every login: scratch targets vanish on preemption, the symlink in HOME does not
    [ -L "$link" ] || { mkdir -p "$(dirname "$link")" && rm -rf "$link" && ln -s "$target" "$link"; }
  done 2>/dev/null
fi
unset u c pair link target
EOF

# Accounts that already exist keep their symlinks in /home across a stop, but the
# scratch targets those point at are gone. Remake them, so only a brand-new account
# still needs a first interactive login.
for h in /home/*; do
  u=${h##*/}; id "$u" &>/dev/null || continue
  for d in "" /go-mod /npm /pnpm; do install -d -o "$u" -g "$u" "/scratch/$u/cache$d"; done
done

# ---------- (c) GitHub runners ----------
# The runner is stateless, so it lives on scratch and is installed from the unit
# rather than from here. systemd retries that step forever, so a failed download
# heals itself; doing it at boot gave a fresh VM no second chance.
echo "$RUNNER_VERSION" > /etc/devbox-runner-version

cat > /usr/local/sbin/install-runner <<'EOF'
#!/bin/bash
# <account>. Root, from the unit. Re-runs whole on every retry, deps included.
set -euo pipefail
DIR=/scratch/$1/runner
V=$(< /etc/devbox-runner-version)
# Stamped with the version and written last: a reboot keeps scratch, so a bare marker
# would pin the old release, and an unstamped one would skip a failed dep step.
[[ "$(cat "$DIR/.installed" 2>/dev/null)" == "$V" ]] && exit 0
rm -rf "$DIR"   # clean tree: extracting over an old release leaves its files behind
install -d -o "$1" -g "$1" "$DIR"
curl -fsSL --retry 3 "https://github.com/actions/runner/releases/download/v$V/actions-runner-linux-x64-$V.tar.gz" | tar xz -C "$DIR"
chown -R "$1:$1" "$DIR"
"$DIR"/bin/installdependencies.sh >/dev/null   # apt is a no-op once satisfied
echo "$V" > "$DIR/.installed"
EOF
chmod +x /usr/local/sbin/install-runner

cat > /usr/local/sbin/register-runner <<'EOF'
#!/bin/bash
# <account> <install-dir> <work-dir>. Runs as the account, every boot.
set -euo pipefail
NAME=$1 DIR=$2 WORK=$3 ORG=bytebase
md() { curl -sf -H "Metadata-Flavor: Google" "http://metadata.google.internal/computeMetadata/v1/$1"; }
PROJECT=$(md project/project-id)
ACCESS=$(md instance/service-accounts/default/token | python3 -c 'import sys,json; print(json.load(sys.stdin)["access_token"])')
PAT=$(curl -sf -H "Authorization: Bearer $ACCESS" \
  "https://secretmanager.googleapis.com/v1/projects/${PROJECT}/secrets/gh-runner-pat/versions/latest:access" \
  | python3 -c 'import sys,base64,json; print(base64.b64decode(json.load(sys.stdin)["payload"]["data"]).decode())')
TOKEN=$(curl -sfX POST -H "Authorization: Bearer $PAT" -H "Accept: application/vnd.github+json" \
  "https://api.github.com/orgs/${ORG}/actions/runners/registration-token" \
  | python3 -c 'import sys,json; print(json.load(sys.stdin)["token"])')
cd "$DIR"
rm -f .runner .credentials .credentials_rsakey   # config.sh refuses to overwrite an existing config
./config.sh --unattended --replace --url "https://github.com/${ORG}" --token "$TOKEN" \
  --name "$(hostname -s)-$NAME" --work "$WORK"   # host-qualified name: never collides with another box
EOF
chmod +x /usr/local/sbin/register-runner

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
ExecStartPre=/usr/local/sbin/register-runner %i /scratch/%i/runner /scratch/%i/work
ExecStart=/scratch/%i/runner/run.sh
# install-runner fetches ~200 MB and waits on the apt lock, which the 90s default
# would kill -- and the retry restarts the download from zero.
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
HIGH=85 CRITICAL=95
find /tmp -maxdepth 1 -name 'go-build*' -type d -mmin +1440 -exec rm -rf {} +
used() { df -P /scratch | awk 'NR==2 {print $5+0}'; }
idle() { ! pgrep -f Runner.Worker >/dev/null && [[ -z "$(ss -Htn state established '( sport = :22 )')" ]]; }

[[ $(used) -ge $HIGH ]] || exit 0
if ! idle && [[ $(used) -lt $CRITICAL ]]; then logger -t cache-gc "deferred: busy at $(used)%"; exit 0; fi
logger -t cache-gc "cleaning: $(used)% used"
find /tmp -mindepth 1 -delete 2>/dev/null   # /tmp is on scratch; nothing is using it
docker system prune -af --volumes >/dev/null 2>&1
docker volume prune -af >/dev/null 2>&1   # system prune takes anonymous volumes only
for d in /scratch/*/; do
  u=${d%/}; u=${u##*/}; id "$u" &>/dev/null || continue
  rm -rf "$d"{cache,home,work}   # runner/ stays: it holds the install and its stamp
  for x in /cache /cache/go-mod /cache/npm /cache/pnpm /home /work; do
    install -d -o "$u" -g "$u" "/scratch/$u$x"   # recreate: symlinks must not dangle
  done
done
logger -t cache-gc "cleaned -> $(used)%"
EOF
chmod +x /usr/local/sbin/cache-gc
# PATH first: cron's default omits /usr/sbin, and a missing `ss` would make idle()
# look true and let the destructive tiers run during a live job.
printf 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\n*/30 * * * * root /usr/local/sbin/cache-gc\n' > /etc/cron.d/devbox

# One outcome check before any runner is advertised. An unset or unreadable
# DockerRootDir catches a failed package install, a dockerd that would not start and a
# daemon.json it rejected; requiring /scratch also catches one that was never written,
# which starts dockerd cleanly on the boot disk. Otherwise: three healthy-looking
# runners that fail every job, or quietly fill the disk holding /home.
docker info --format '{{.DockerRootDir}}' 2>/dev/null | grep -q '^/scratch/' \
  || { logger -t devbox-startup "FATAL: docker unusable or not on Local SSD"; exit 1; }

systemctl daemon-reload
systemctl restart actions-runner@runner1 actions-runner@runner2 actions-runner@runner3

# Last, and time-bounded: guest memory and swap are invisible without the Ops Agent,
# but it is observability. A stall here must not hold up CI capacity.
systemctl is-active --quiet google-cloud-ops-agent \
  || { curl -sS --retry 3 --max-time 120 -o /tmp/ops.sh https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh \
       && timeout 300 bash /tmp/ops.sh --also-install; } \
  || logger -t devbox-startup "WARN: Ops Agent install failed; no memory or swap metrics"   # retried next boot
```

## 3. Considered

**N2D** — its guaranteed Milan silicon avoids N2's Cascade-Lake-or-Ice-Lake lottery,
but no region prices it close to Toronto N2. Spot rates are set per family *per
region*, so there is no cheap region, only cheap region-and-family pairs.

**Arm (C4A, T2A)** — not every testcontainer image ships arm64, and
`mcr.microsoft.com/mssql/server` is one that does not. It is also not the saving it
looks like: Arm has no custom shapes, so the nearest fit costs more for fewer cores.

**48 GB or 64 GB RAM** — a comparable box runs two runner slots in 16 GB plus swap
without an OOM kill, so 32 GB already doubles what has sufficed. Machine type is a
stopped-instance change, so erring low costs one restart.

**On-demand instead of Spot** — about 2.5x per hour. Reversible on a stopped instance
with `set-scheduling`.

**Two machines instead of one** — cheaper for the same work, because a combined box
stays up for the union of both duty cycles at the sum of both sizes. Consolidation was
chosen for operational simplicity.

**750 GB Local SSD** — another $19.80/mo, against a projected 49% use of 375 GB. Local
SSD is the one capacity fixed at creation, so it is the first thing to revisit if that
projection is wrong.

**A firewall rule and network tag** — the default VPC leaves several ports open to
`0.0.0.0/0`, but nothing here listens on them. Tags can be added to a running instance,
so a rule can follow later.

**Repository-level runner registration** — it would bound a stolen credential to one
repo, but its endpoints need the far broader *Administration* permission, and it would
not give org-wide runners.

**A Cloud Run service minting JIT configs** — it would keep the PAT off the box
entirely, but it is too much infrastructure for a dev box. The accepted cost: any CI
job can read the PAT from the metadata server, register its own runner, and keep
taking org jobs until that registration is deleted. Neither obvious bound actually
binds -- a runner credential outlives the PAT that created it, and a rogue host picks
its own `--name`, so alerting on names outside `devbox-*` catches mistakes rather than
an attacker. Removing a registration is the control; the alert is a convenience.

**Rootless Docker** — the socket is world-accessible, which is how an OS Login account
the script cannot name gets Docker. That is root-equivalent, and accepted. Be clear
what that accepts: CI runs pull-request code with the same reach, so a job can read or
change any `/home` on the box -- uncommitted work, agent CLI state, `~/.ssh`. Treat
this as a shared machine and keep nothing on it you would not put on one.

**Pinning Ice Lake** — deferred. `cpuPlatform` after each restart shows what Toronto
hands out. A pin, or a different region, is a stop-and-start away.

## Reference

- Local SSD: https://cloud.google.com/compute/docs/disks/local-ssd
- Self-hosted runners: https://docs.github.com/en/actions/hosting-your-own-runners
- Fine-grained PAT permissions: https://docs.github.com/en/rest/authentication/permissions-required-for-fine-grained-personal-access-tokens
- CPU platforms: https://cloud.google.com/compute/docs/cpu-platforms
