#!/bin/bash
# Creates the devbox: the SSH target for agentic coding tools and the host for the
# org's self-hosted GitHub Actions runners. Design: docs/design/devbox-and-ci-runner.md
#
# Run once, as a project Owner: it sets IAM policy on the project and on the secret,
# which Editor cannot do. It stops at the first failed step and is not idempotent --
# after a partial run, delete what was created before re-running.
#
# Prerequisite, and the one step no API can do: a fine-grained GitHub PAT owned by the
# `bytebase` org, with organization permission "Self-hosted runners: Read and write"
# and nothing else. See the Create section of the design doc.
set -euo pipefail
PROJECT=bytebase-dev
ZONE=northamerica-northeast2-a
SA=devbox@${PROJECT}.iam.gserviceaccount.com
HERE=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)

gcloud services enable secretmanager.googleapis.com oslogin.googleapis.com \
  monitoring.googleapis.com logging.googleapis.com --project=$PROJECT

# Service account; its roles are granted below, nothing more.
gcloud iam service-accounts create devbox --project=$PROJECT \
  --display-name="devbox: dev host and CI runners"

# The PAT, readable by that service account and nothing else.
read -rsp 'GitHub PAT: ' GH_PAT; echo   # not `&& echo`: set -e ignores a failed read inside an && list
printf '%s' "$GH_PAT" | gcloud secrets create gh-runner-pat \
  --project=$PROJECT --replication-policy=automatic --data-file=-
unset GH_PAT

gcloud secrets add-iam-policy-binding gh-runner-pat --project=$PROJECT \
  --member="serviceAccount:${SA}" --role=roles/secretmanager.secretAccessor

# Metrics and logs from the Ops Agent. Without these the agent gets 403s and guest
# memory never reaches Cloud Monitoring -- a scope is not a role.
for r in monitoring.metricWriter logging.logWriter; do
  gcloud projects add-iam-policy-binding $PROJECT \
    --member="serviceAccount:${SA}" --role=roles/$r --condition=None
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
  --local-ssd=interface=NVME --local-ssd=interface=NVME \
  `# two partitions, not one: N2 at 20 vCPU accepts only 0, 2, 4, 8, 16 or 24` \
  --service-account=$SA --scopes=cloud-platform \
  --metadata=enable-oslogin=TRUE \
  --metadata-from-file=startup-script="$HERE/startup.sh"
