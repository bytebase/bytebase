<template>
  <div class="w-full px-4 flex flex-col gap-y-6 py-4">
    <!-- Section 1: What is GitOps -->
    <div class="border border-gray-200 rounded-lg p-6 flex flex-col gap-y-2">
      <h2 class="text-lg font-medium">
        {{ $t("gitops.overview.title") }}
      </h2>
      <div>
        <p class="textinfolabel">
          {{ $t("gitops.overview.description") }}
        </p>
        <p class="textinfolabel">
          {{ $t("gitops.overview.description-git") }}
        </p>
      </div>
      <img
        :src="gitopsWorkflowImage"
        alt="GitOps Workflow"
        class="w-full max-w-4xl object-contain my-2"
      />
      <!-- Documentation link -->
      <div>
        <a
          href="https://docs.bytebase.com/vcs-integration/overview?source=console"
          target="_blank"
          class="text-accent hover:underline"
        >
          {{ $t("gitops.documentation") }} &rarr;
        </a>
      </div>
    </div>

    <!-- Section 2: Checks before we start -->
    <div class="border border-gray-200 rounded-lg p-6 flex flex-col gap-y-1">
      <h2 class="text-lg font-medium mb-2">
        {{ $t("gitops.checklist.title") }}
      </h2>

      <!-- Check 1: External URL -->
      <div class="flex items-start gap-x-3 py-3">
        <CheckIcon
          v-if="bytebaseUrl"
          class="w-5 h-5 text-success shrink-0"
        />
        <XCircleIcon
          v-else
          class="w-5 h-5 text-warning shrink-0"
        />

        <div class="flex flex-col gap-y-1">
          <span class="text-sm font-medium">
            {{ $t("gitops.checklist.external-url") }}
          </span>
          <span v-if="bytebaseUrl" class="text-sm text-control-light">
            {{ bytebaseUrl }}
          </span>
          <MissingExternalURLAttention class="mt-1" />
        </div>
      </div>

      <!-- Check 2: Workload Identity -->
      <div class="flex items-start gap-x-3 py-3">
        <CheckIcon
          v-if="selectedIdentityEmail"
          class="w-5 h-5 text-success shrink-0"
        />
        <XCircleIcon
          v-else
          class="w-5 h-5 text-warning shrink-0"
        />
        <div class="flex flex-col gap-y-2 flex-1">
          <span class="text-sm font-medium">{{
            $t("gitops.checklist.workload-identity")
          }}</span>
          <div class="flex items-center gap-x-3">
            <NSelect
              v-model:value="selectedIdentityEmail"
              :options="identityOptions"
              :placeholder="
                $t('gitops.workload-identity.select-placeholder')
              "
              clearable
              class="max-w-md"
            />
            <PermissionGuardWrapper
              v-slot="slotProps"
              :project="project"
              :permissions="[
                'bb.workloadIdentities.create'
              ]"
            >
              <NButton :disabled="slotProps.disabled" @click="showCreateDrawer = true">
                {{ $t("common.create") }}
              </NButton>
            </PermissionGuardWrapper>
          </div>
          <p
            v-if="identityOptions.length === 0 && !isLoading"
            class="text-sm text-control-light"
          >
            {{ $t("gitops.workload-identity.no-identity") }}
          </p>
          <a
            v-if="repoUrl"
            :href="repoUrl"
            target="_blank"
            class="text-sm text-accent hover:underline"
          >
            {{ repoUrl }} &rarr;
          </a>
        </div>
      </div>

      <!-- Check 3: Target Databases -->
      <div class="flex items-start gap-x-3 py-3">
        <CheckIcon
          v-if="selectedDatabaseNames.length > 0"
          class="w-5 h-5 text-success shrink-0"
        />
        <XCircleIcon
          v-else
          class="w-5 h-5 text-warning shrink-0"
        />
        <div class="flex flex-col gap-y-2 flex-1">
          <span class="text-sm font-medium">{{
            $t("gitops.checklist.target-databases")
          }}</span>
          <div class="max-w-md">
            <DatabaseSelect
              v-model:value="selectedDatabaseNames"
              :project-name="projectName"
              :multiple="true"
            />
          </div>
          <p
            v-if="targetsString"
            class="text-sm text-control-light"
          >
            <span class="font-medium">targets:</span>
            <code class="text-xs bg-gray-100 px-1 py-0.5 rounded">{{
              targetsString
            }}</code>
          </p>
        </div>
      </div>
    </div>

    <!-- Section 3: Workflow file generation -->
    <div class="border border-gray-200 rounded-lg p-6 flex flex-col gap-y-3">
      <h2 class="text-lg font-medium">
        {{ $t("gitops.workflow.title") }}
      </h2>
      <p class="text-sm text-control-light">
        {{ $t("gitops.workflow.description") }}
      </p>

      <NTabs v-model:value="activeTab" type="line" animated>
        <NTabPane name="github-actions" tab="GitHub Actions">
          <!-- Runner type toggle -->
          <div class="flex items-center gap-x-2 mt-3">
            <span class="text-sm">Self-hosted</span>
            <NSwitch v-model:value="useSelfhostRunner" />
          </div>
          <!-- sql-review.yml -->
          <div class="flex flex-col gap-y-2 mt-3">
            <div class="flex items-center gap-x-2">
              <button
                class="flex items-center gap-x-1 text-sm text-control-light hover:text-control cursor-pointer"
                @click="showSqlReviewYaml = !showSqlReviewYaml"
              >
                <ChevronRightIcon
                  v-if="!showSqlReviewYaml"
                  class="w-4 h-4"
                />
                <ChevronDownIcon v-else class="w-4 h-4" />
                {{
                  $t("gitops.workflow.file-hint", {
                    filePath: ".github/workflows/sql-review.yml",
                  })
                }}
              </button>
              <CopyButton :content="githubSqlReviewYaml" />
            </div>
            <NInput
              v-show="showSqlReviewYaml"
              :value="githubSqlReviewYaml"
              type="textarea"
              readonly
              :autosize="{ minRows: 10, maxRows: 30 }"
              class="font-mono text-sm"
            />
          </div>
          <!-- release.yml -->
          <div class="flex flex-col gap-y-2 mt-4">
            <div class="flex items-center gap-x-2">
              <button
                class="flex items-center gap-x-1 text-sm text-control-light hover:text-control cursor-pointer"
                @click="showReleaseYaml = !showReleaseYaml"
              >
                <ChevronRightIcon
                  v-if="!showReleaseYaml"
                  class="w-4 h-4"
                />
                <ChevronDownIcon v-else class="w-4 h-4" />
                {{
                  $t("gitops.workflow.file-hint", {
                    filePath: ".github/workflows/release.yml",
                  })
                }}
              </button>
              <CopyButton :content="githubReleaseYaml" />
            </div>
            <NInput
              v-show="showReleaseYaml"
              :value="githubReleaseYaml"
              type="textarea"
              readonly
              :autosize="{ minRows: 10, maxRows: 30 }"
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>
        <NTabPane name="gitlab-ci" tab="GitLab CI">
          <div class="flex flex-col gap-y-2 mt-3">
            <div class="flex items-center gap-x-2">
              <button
                class="flex items-center gap-x-1 text-sm text-control-light hover:text-control cursor-pointer"
                @click="showGitlabCiYaml = !showGitlabCiYaml"
              >
                <ChevronRightIcon
                  v-if="!showGitlabCiYaml"
                  class="w-4 h-4"
                />
                <ChevronDownIcon v-else class="w-4 h-4" />
                {{
                  $t("gitops.workflow.file-hint", {
                    filePath: ".gitlab-ci.yml",
                  })
                }}
              </button>
              <CopyButton :content="gitlabCiYaml" />
            </div>
            <NInput
              v-show="showGitlabCiYaml"
              :value="gitlabCiYaml"
              type="textarea"
              readonly
              :autosize="{ minRows: 10, maxRows: 30 }"
              class="font-mono text-sm"
            />
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <!-- Section 4: Test your first GitOps migration -->
    <div class="border border-gray-200 rounded-lg p-6 flex flex-col gap-y-4">
      <h2 class="text-lg font-medium">
        {{ $t("gitops.test-setup.title") }}
      </h2>
      <p class="text-sm text-control-light">
        {{ $t("gitops.test-setup.description") }}
      </p>

      <!-- Sample migration file -->
      <div class="flex flex-col gap-y-2">
        <div class="flex items-center gap-x-2">
          <span class="text-sm text-control-light">{{ sampleFilePath }}</span>
          <CopyButton :content="sampleSql" />
        </div>
        <NInput
          :value="sampleSql"
          type="textarea"
          readonly
          :autosize="{ minRows: 5, maxRows: 15 }"
          class="font-mono text-sm"
        />
      </div>

      <!-- Step-by-step instructions -->
      <div class="flex flex-col gap-y-3">
        <div class="flex items-start gap-x-3">
          <span
            class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-gray-200 text-gray-600 text-xs shrink-0 mt-0.5"
          >
            1
          </span>
          <p class="text-sm text-control-light">
            {{
              $t("gitops.test-setup.step-create-branch", {
                branch: branch,
              })
            }}
          </p>
        </div>
        <div class="flex items-start gap-x-3">
          <span
            class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-gray-200 text-gray-600 text-xs shrink-0 mt-0.5"
          >
            2
          </span>
          <p class="text-sm text-control-light">
            {{ $t("gitops.test-setup.step-sql-review") }}
          </p>
        </div>
        <div class="flex items-start gap-x-3">
          <span
            class="inline-flex items-center justify-center w-5 h-5 rounded-full bg-gray-200 text-gray-600 text-xs shrink-0 mt-0.5"
          >
            3
          </span>
          <p class="text-sm text-control-light">
            {{ $t("gitops.test-setup.step-merge") }}
          </p>
        </div>
      </div>

      <!-- Naming convention hint -->
      <p class="text-sm text-control-light">
        {{ $t("gitops.test-setup.naming-convention") }}
      </p>
    </div>
  </div>

  <CreateWorkloadIdentityDrawer
    v-if="showCreateDrawer"
    :project="projectName"
    @close="showCreateDrawer = false"
    @created="handleWorkloadIdentityCreated"
  />
</template>

<script lang="ts" setup>
import {
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  XCircleIcon,
} from "lucide-vue-next";
import { NButton, NInput, NSelect, NSwitch, NTabPane, NTabs } from "naive-ui";
import { computed, onMounted, ref, watch } from "vue";
import gitopsWorkflowImage from "@/assets/gitops-workflow.svg";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import CreateWorkloadIdentityDrawer from "@/components/User/Settings/CreateWorkloadIdentityDrawer.vue";
import { CopyButton, DatabaseSelect } from "@/components/v2";
import { MissingExternalURLAttention } from "@/components/v2/Form";
import { useActuatorV1Store, useProjectByName } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useWorkloadIdentityStore } from "@/store/modules/workloadIdentity";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { WorkloadIdentityConfig_ProviderType } from "@/types/proto-es/v1/user_service_pb";
import type { WorkloadIdentity } from "@/types/proto-es/v1/workload_identity_service_pb";

const props = defineProps<{
  projectId: string;
}>();

const actuatorStore = useActuatorV1Store();
const workloadIdentityStore = useWorkloadIdentityStore();

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const { project } = useProjectByName(projectName);

const showCreateDrawer = ref(false);
const selectedIdentityEmail = ref<string | null>(null);
const selectedDatabaseNames = ref<string[]>([]);
const isLoading = ref(false);
const identityOptions = ref<{ label: string; value: string }[]>([]);
const identityMap = ref<Map<string, WorkloadIdentity>>(new Map());
const activeTab = ref("github-actions");
const useSelfhostRunner = ref(false);
const showSqlReviewYaml = ref(false);
const showReleaseYaml = ref(false);
const showGitlabCiYaml = ref(false);

const selectedIdentity = computed(() => {
  if (!selectedIdentityEmail.value) return undefined;
  return identityMap.value.get(selectedIdentityEmail.value);
});

const selectedConfig = computed(
  () => selectedIdentity.value?.workloadIdentityConfig
);

// Parse subject pattern to extract owner/repo/branch.
const parsedSubject = computed(() => {
  const pattern = selectedConfig.value?.subjectPattern;
  if (!pattern) return undefined;

  const providerType = selectedConfig.value?.providerType;

  if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
    const match = pattern.match(/^repo:([^/]+)\/(.*)$/);
    if (!match) return undefined;
    const owner = match[1];
    const rest = match[2];
    if (rest === "*") return { owner, repo: "", branch: "" };
    const repoMatch = rest.match(/^([^:]+):(.*)$/);
    if (!repoMatch) return undefined;
    const repo = repoMatch[1];
    const refPart = repoMatch[2];
    if (refPart === "*") return { owner, repo, branch: "" };
    const branchMatch = refPart.match(/^ref:refs\/heads\/(.+)$/);
    return { owner, repo, branch: branchMatch?.[1] ?? "" };
  }

  if (providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    const match = pattern.match(/^project_path:([^/]+)\/(.*)$/);
    if (!match) return undefined;
    const owner = match[1];
    const rest = match[2];
    if (rest === "*") return { owner, repo: "", branch: "" };
    const projectMatch = rest.match(/^([^:]+):(.*)$/);
    if (!projectMatch) return undefined;
    const repo = projectMatch[1];
    const refPart = projectMatch[2];
    if (refPart === "*") return { owner, repo, branch: "" };
    const refTypeMatch = refPart.match(/^ref_type:(?:branch|tag):ref:(.+)$/);
    return { owner, repo, branch: refTypeMatch?.[1] ?? "" };
  }

  return undefined;
});

// Build a clickable repository URL from the identity.
const repoUrl = computed(() => {
  const parsed = parsedSubject.value;
  if (!parsed?.owner || !parsed.repo) return "";

  const providerType = selectedConfig.value?.providerType;
  if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
    return `https://github.com/${parsed.owner}/${parsed.repo}`;
  }
  if (providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    const issuer = selectedConfig.value?.issuerUrl ?? "https://gitlab.com";
    const base = issuer.replace(/\/$/, "");
    return `${base}/${parsed.owner}/${parsed.repo}`;
  }
  return "";
});

// Branch from the identity, fallback to "main".
const branch = computed(() => parsedSubject.value?.branch || "main");

// Auto-switch tab when identity changes.
watch(selectedConfig, (config) => {
  if (!config) return;
  if (config.providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    activeTab.value = "gitlab-ci";
  } else {
    activeTab.value = "github-actions";
  }
});

const bytebaseUrl = computed(() => {
  const url = actuatorStore.serverInfo?.externalUrl ?? "";
  return url.replace(/\/$/, "");
});

const workloadIdentityEmail = computed(() => {
  if (!selectedIdentityEmail.value) {
    return "{WORKLOAD_IDENTITY_EMAIL}";
  }
  return selectedIdentityEmail.value;
});

const targetsString = computed(() => {
  if (selectedDatabaseNames.value.length === 0) {
    return "";
  }
  return selectedDatabaseNames.value.join(",");
});

const targetsPlaceholder = computed(() => {
  return targetsString.value || "instances/{instance}/databases/{database}";
});

const runsOn = computed(() => {
  return useSelfhostRunner.value ? "self-hosted" : "ubuntu-latest";
});

const sampleFilePath = "migrations/20240101000000_create_sample_table.sql";

const sampleSql = `CREATE TABLE sample (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);`;

const githubSqlReviewYaml = computed(() => {
  return `name: SQL Review
on:
  pull_request:
    branches: ["${branch.value}"]
    paths: ["migrations/*.sql"]
jobs:
  sql-review:
    permissions:
      id-token: write
      pull-requests: write
    runs-on: ${runsOn.value}
    container:
      image: bytebase/bytebase-action
    env:
      BYTEBASE_URL: ${bytebaseUrl.value}
      BYTEBASE_WORKLOAD_IDENTITY: ${workloadIdentityEmail.value}
    steps:
      - uses: actions/checkout@v4
      - name: Exchange token
        id: bytebase-auth
        run: |
          OIDC_TOKEN=$(curl -s -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \\
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=bytebase" | jq -r '.value')
          ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
            -H "Content-Type: application/json" \\
            -d "{\\"token\\":\\"$OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
            | jq -r '.accessToken')
          echo "access-token=$ACCESS_TOKEN" >> $GITHUB_OUTPUT
      - name: SQL Review
        env:
          GITHUB_TOKEN: \${{ secrets.GITHUB_TOKEN }}
        run: |
          bytebase-action check \\
            --url=$BYTEBASE_URL \\
            --access-token=\${{ steps.bytebase-auth.outputs.access-token }} \\
            --project=projects/${props.projectId} \\
            --targets=${targetsPlaceholder.value} \\
            --file-pattern=migrations/*.sql`;
});

const exchangeTokenStep = `      - name: Exchange token
        id: bytebase-auth
        run: |
          OIDC_TOKEN=$(curl -s -H "Authorization: bearer $ACTIONS_ID_TOKEN_REQUEST_TOKEN" \\
            "$ACTIONS_ID_TOKEN_REQUEST_URL&audience=bytebase" | jq -r '.value')
          ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
            -H "Content-Type: application/json" \\
            -d "{\\"token\\":\\"$OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
            | jq -r '.accessToken')
          echo "access-token=$ACCESS_TOKEN" >> $GITHUB_OUTPUT`;

const accessTokenFlag =
  "--access-token=${{ steps.bytebase-auth.outputs.access-token }}";

const githubReleaseYaml = computed(() => {
  return `name: Rollout
on:
  push:
    branches: ["${branch.value}"]
    paths: ["migrations/*.sql"]
env:
  BYTEBASE_URL: ${bytebaseUrl.value}
  BYTEBASE_WORKLOAD_IDENTITY: ${workloadIdentityEmail.value}
  BYTEBASE_PROJECT: projects/${props.projectId}
jobs:
  build:
    runs-on: ${runsOn.value}
    steps:
      - uses: actions/checkout@v4
      - name: Build
        run: echo "Building..."
  create-rollout:
    needs: build
    permissions:
      id-token: write
    runs-on: ${runsOn.value}
    container:
      image: bytebase/bytebase-action
    outputs:
      bytebase-plan: \${{ steps.set-output.outputs.plan }}
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep}
      - name: Create rollout
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --targets=${targetsPlaceholder.value} \\
            --file-pattern=migrations/*.sql \\
            --output=\${{ runner.temp }}/bytebase-metadata.json
      - name: Set output
        id: set-output
        run: |
          PLAN=$(jq -r .plan \${{ runner.temp }}/bytebase-metadata.json)
          echo "plan=$PLAN" >> $GITHUB_OUTPUT
  deploy-to-test:
    needs: create-rollout
    permissions:
      id-token: write
    runs-on: ${runsOn.value}
    environment: test
    container:
      image: bytebase/bytebase-action
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep}
      - name: Deploy to test
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/test \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}
  deploy-to-prod:
    needs: [deploy-to-test, create-rollout]
    permissions:
      id-token: write
    runs-on: ${runsOn.value}
    environment: prod
    container:
      image: bytebase/bytebase-action
    steps:
      - uses: actions/checkout@v4
${exchangeTokenStep}
      - name: Deploy to prod
        run: |
          bytebase-action rollout \\
            --url=$BYTEBASE_URL \\
            ${accessTokenFlag} \\
            --project=$BYTEBASE_PROJECT \\
            --target-stage=environments/prod \\
            --plan=\${{ needs.create-rollout.outputs.bytebase-plan }}`;
});

const gitlabExchangeScript = `    - |
      ACCESS_TOKEN=$(curl -s -X POST "$BYTEBASE_URL/v1/auth:exchangeToken" \\
        -H "Content-Type: application/json" \\
        -d "{\\"token\\":\\"$CI_JOB_JWT_V2\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
        | jq -r '.accessToken')
      export BYTEBASE_ACCESS_TOKEN=$ACCESS_TOKEN`;

const gitlabCiYaml = computed(() => {
  return `stages:
  - sql-review
  - create-rollout
  - deploy-to-test
  - deploy-to-prod

variables:
  BYTEBASE_URL: ${bytebaseUrl.value}
  BYTEBASE_WORKLOAD_IDENTITY: ${workloadIdentityEmail.value}
  BYTEBASE_PROJECT: projects/${props.projectId}
  BYTEBASE_TARGETS: ${targetsPlaceholder.value}
  FILE_PATTERN: "migrations/*.sql"

sql-review:
  stage: sql-review
  image: bytebase/bytebase-action
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
  script:
${gitlabExchangeScript}
    - bytebase-action check --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN

create-rollout:
  stage: create-rollout
  image: bytebase/bytebase-action
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_COMMIT_BRANCH == "${branch.value}"
  script:
${gitlabExchangeScript}
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --targets=$BYTEBASE_TARGETS --file-pattern=$FILE_PATTERN --output=bytebase-metadata.json
  artifacts:
    paths: [bytebase-metadata.json]

deploy-to-test:
  stage: deploy-to-test
  image: bytebase/bytebase-action
  needs: [create-rollout]
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_COMMIT_BRANCH == "${branch.value}"
  environment: test
  script:
${gitlabExchangeScript}
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --target-stage=environments/test --plan=$PLAN

deploy-to-prod:
  stage: deploy-to-prod
  image: bytebase/bytebase-action
  needs: [deploy-to-test]
  id_tokens:
    GITLAB_OIDC_TOKEN:
      aud: bytebase
  rules:
    - if: $CI_COMMIT_BRANCH == "${branch.value}"
  environment: prod
  when: manual
  script:
${gitlabExchangeScript}
    - PLAN=$(jq -r .plan bytebase-metadata.json)
    - bytebase-action rollout --url=$BYTEBASE_URL --access-token=$BYTEBASE_ACCESS_TOKEN --project=$BYTEBASE_PROJECT --target-stage=environments/prod --plan=$PLAN`;
});

const fetchWorkloadIdentities = async () => {
  isLoading.value = true;
  try {
    const response = await workloadIdentityStore.listWorkloadIdentities({
      parent: projectName.value,
      pageSize: 100,
      pageToken: undefined,
      showDeleted: false,
    });
    const map = new Map<string, WorkloadIdentity>();
    const options: { label: string; value: string }[] = [];
    for (const wi of response.workloadIdentities) {
      map.set(wi.email, wi);
      options.push({
        label: `${wi.title} (${wi.email})`,
        value: wi.email,
      });
    }
    identityMap.value = map;
    identityOptions.value = options;
  } finally {
    isLoading.value = false;
  }
};

const handleWorkloadIdentityCreated = async (user: User) => {
  await fetchWorkloadIdentities();
  selectedIdentityEmail.value = user.email;
};

onMounted(() => {
  fetchWorkloadIdentities();
});
</script>
