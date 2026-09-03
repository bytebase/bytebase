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

Developers already have access: the developer group holds Editor on the project,
which covers OS Login with `sudo` and acting as the service account.

**2. Everything else.** Run as a project Owner: it sets IAM policy on the project and
on the secret, which Editor cannot do. It stops at the first failed step.

```bash
set -e
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
  --metadata-from-file=startup-script=scripts/devbox-startup.sh
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

Run `gcloud compute config-ssh --project=bytebase-dev` once. It writes the `~/.ssh/config` entries, and this
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

Passed to the create command above. It runs as root on every
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
the runner's, and each start installs the latest release.

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

Lives at [`scripts/devbox-startup.sh`](../../scripts/devbox-startup.sh) — one copy, so it
cannot drift from what the instance actually runs. Read it there.

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
