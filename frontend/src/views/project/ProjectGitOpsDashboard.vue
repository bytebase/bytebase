<template>
  <div class="w-full px-4 flex flex-col gap-y-1 py-4">
    <!-- Section 1: What is GitOps -->
    <div class="border border-gray-200 rounded p-6 flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <h2 class="text-lg font-medium">
          {{ $t("gitops.overview.title") }}
        </h2>
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

    <span
      class="mx-auto w-0.5 h-8 bg-block-border"
      aria-hidden="true"
    ></span>

    <!-- Section 2: Checks before we start -->
    <div class="border border-gray-200 rounded p-6 flex flex-col gap-y-3">
      <h2 class="text-lg font-medium">
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

        <div class="flex flex-col flex-1 gap-y-1">
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
          v-if="selectedIdentityName"
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
            <WorkloadIdentitySelect
              v-model:value="selectedIdentityName"
              :parent="projectName"
              :placeholder="
                $t('gitops.workload-identity.select-placeholder')
              "
              clearable
              class="max-w-lg"
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
          v-if="hasTargetSelected"
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
          <NRadioGroup
            :value="targetTab"
            size="small"
            @update:value="handleTargetTabChange"
          >
            <NRadio value="GROUP">
              {{ $t("common.database-group") }}
            </NRadio>
            <NRadio value="DATABASE">
              {{ $t("common.databases") }}
            </NRadio>
          </NRadioGroup>
          <div class="max-w-lg">
            <template v-if="targetTab === 'GROUP'">
              <NSelect
                :value="selectedDatabaseGroupName"
                :options="dbGroupOptions"
                :placeholder="$t('database-group.select')"
                clearable
                @update:value="selectedDatabaseGroupName = $event"
              />
              <p class="text-xs text-control-light mt-1">
                {{ $t("gitops.checklist.database-group-recommendation") }}
              </p>
            </template>
            <DatabaseSelect
              v-else
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

    <span
      class="mx-auto w-0.5 h-8 bg-block-border"
      aria-hidden="true"
    ></span>

    <!-- Section 3: Workflow file generation -->
    <div class="border border-gray-200 rounded p-6 flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <h2 class="text-lg font-medium">
          {{ $t("gitops.workflow.title") }}
        </h2>
        <p class="text-sm text-control-light">
          {{ $t("gitops.workflow.description") }}
        </p>
      </div>

      <NTabs v-model:value="activeTab" type="line" animated>
        <NTabPane :name="WorkloadIdentityConfig_ProviderType.GITHUB" tab="GitHub Actions">
          <BBAttention v-if="selectedConfig && activeTab !== selectedConfig?.providerType" :type="'error'">
            {{ $t("gitops.workflow.provider-not-match", { provider: getWorkloadIdentityProviderText(selectedConfig.providerType) }) }}
          </BBAttention>
          <!-- Runner type toggle -->
          <div class="flex items-center gap-x-2 my-3">
            <span class="text-sm">
              {{ $t("gitops.workflow.self-hosted-runner") }}
            </span>
            <NSwitch v-model:value="useSelfhostRunner" />
          </div>
          <!-- sql-review.yml -->
          <div class="flex flex-col gap-y-2">
            <div>
              <NButton quaternary size="small" @click="showSqlReviewYaml = !showSqlReviewYaml">
                <template #icon>
                  <ChevronRightIcon
                    v-if="!showSqlReviewYaml"
                    class="w-4 h-4"
                  />
                  <ChevronDownIcon v-else class="w-4 h-4" />
                </template>
                <i18n-t keypath="gitops.workflow.file-hint">
                  <template #filePath>
                    <span class="font-bold mx-1">
                      .github/workflows/sql-review.yml
                    </span>
                  </template>
                  <template #repository>
                    <span class="font-bold mx-1">
                      {{ parsedSubject?.repo }}
                    </span>
                  </template>
                </i18n-t>
              </NButton>
            </div>
            <div v-show="showSqlReviewYaml" class="relative rounded-xs p-4 bg-gray-50">
              <div class="absolute top-2 right-2 p-2">
                <CopyButton size="medium" :content="githubSqlReviewYaml" />
              </div>
              <NConfigProvider :hljs="hljs">
                <NCode language="yaml" :code="githubSqlReviewYaml" />
              </NConfigProvider>
            </div>
          </div>
          <!-- release.yml -->
          <div class="flex flex-col gap-y-2 mt-4">
            <div>
              <NButton quaternary size="small" @click="showReleaseYaml = !showReleaseYaml">
                <template #icon>
                  <ChevronRightIcon
                    v-if="!showReleaseYaml"
                    class="w-4 h-4"
                  />
                  <ChevronDownIcon v-else class="w-4 h-4" />
                </template>
                <i18n-t keypath="gitops.workflow.file-hint">
                  <template #filePath>
                    <span class="font-bold mx-1">
                      .github/workflows/release.yml
                    </span>
                  </template>
                  <template #repository>
                    <span class="font-bold mx-1">
                      {{ parsedSubject?.repo }}
                    </span>
                  </template>
                </i18n-t>
              </NButton>
            </div>
            <div v-show="showReleaseYaml" class="relative rounded-xs p-4 bg-gray-50">
              <div class="absolute top-2 right-2 p-2">
                <CopyButton size="medium" :content="githubReleaseYaml" />
              </div>
              <NConfigProvider :hljs="hljs">
                <NCode language="yaml" :code="githubReleaseYaml" />
              </NConfigProvider>
            </div>
          </div>
        </NTabPane>
        <NTabPane :name="WorkloadIdentityConfig_ProviderType.GITLAB" tab="GitLab CI">
          <div class="flex flex-col gap-y-2">
            <BBAttention v-if="selectedConfig && activeTab !== selectedConfig?.providerType" :type="'error'">
              {{ $t("gitops.workflow.provider-not-match", { provider: getWorkloadIdentityProviderText(selectedConfig.providerType) }) }}
            </BBAttention>
            <div>
              <NButton quaternary size="small" @click="showGitlabCiYaml = !showGitlabCiYaml">
                <template #icon>
                  <ChevronRightIcon
                    v-if="!showGitlabCiYaml"
                    class="w-4 h-4"
                  />
                  <ChevronDownIcon v-else class="w-4 h-4" />
                </template>
                <i18n-t keypath="gitops.workflow.file-hint">
                  <template #filePath>
                    <span class="font-bold mx-1">
                      .gitlab-ci.yml
                    </span>
                  </template>
                  <template #repository>
                    <span class="font-bold mx-1">
                      {{ parsedSubject?.repo }}
                    </span>
                  </template>
                </i18n-t>
              </NButton>
            </div>
            <div v-show="showGitlabCiYaml" class="relative rounded-xs p-4 bg-gray-50">
              <div class="absolute top-2 right-2 p-2">
                <CopyButton size="medium" :content="gitlabCiYaml" />
              </div>
              <NConfigProvider :hljs="hljs">
                <NCode language="yaml" :code="gitlabCiYaml" />
              </NConfigProvider>
            </div>
          </div>
        </NTabPane>
        <NTabPane name="BITBUCKET" tab="Bitbucket Pipelines">
          <div class="flex flex-col items-center justify-center py-8 text-control-light">
            <span class="text-sm">
              {{ $t("gitops.workflow.examples-coming-soon", { provider: "Bitbucket" }) }}
            </span>
            <p class="text-xs mt-1">
              {{ $t("gitops.workflow.wif-not-available", { provider: "Bitbucket" }) }}
            </p>
          </div>
        </NTabPane>
        <NTabPane name="AZURE_DEVOPS" tab="Azure DevOps">
          <div class="flex flex-col items-center justify-center py-8 text-control-light">
            <span class="text-sm">
              {{ $t("gitops.workflow.examples-coming-soon", { provider: "Azure DevOps" }) }}
            </span>
            <p class="text-xs mt-1">
              {{ $t("gitops.workflow.wif-not-available", { provider: "Azure DevOps" }) }}
            </p>
          </div>
        </NTabPane>
      </NTabs>
    </div>

    <span
      class="mx-auto w-0.5 h-8 bg-block-border"
      aria-hidden="true"
    ></span>

    <!-- Section 4: Test your first GitOps migration -->
    <div class="border border-gray-200 rounded p-6 flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <h2 class="text-lg font-medium">
          {{ $t("gitops.test-setup.title") }}
        </h2>
        <p class="text-sm text-control-light">
          {{ $t("gitops.test-setup.description") }}
        </p>
      </div>

      <!-- Sample migration file -->
      <div class="flex flex-col gap-y-2">
        <div class="flex items-center gap-x-2">
          <span class="text-sm text-control-light font-bold">{{ sampleFilePath }}</span>
        </div>
        <div class="relative rounded-xs p-4 bg-gray-50">
          <div class="absolute top-2 right-2 p-2">
            <CopyButton size="medium" :content="sampleSql" />
          </div>
          <NConfigProvider :hljs="hljs">
            <NCode language="sql" :code="sampleSql" />
          </NConfigProvider>
        </div>
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
import hljs from "highlight.js/lib/core";
import {
  CheckIcon,
  ChevronDownIcon,
  ChevronRightIcon,
  XCircleIcon,
} from "lucide-vue-next";
import {
  NButton,
  NCode,
  NConfigProvider,
  NRadio,
  NRadioGroup,
  NSelect,
  NSwitch,
  NTabPane,
  NTabs,
} from "naive-ui";
import { computed, ref, watch } from "vue";
import gitopsWorkflowImage from "@/assets/gitops-workflow.svg";
import { BBAttention } from "@/bbkit";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import CreateWorkloadIdentityDrawer from "@/components/User/Settings/CreateWorkloadIdentityDrawer.vue";
import {
  CopyButton,
  DatabaseSelect,
  WorkloadIdentitySelect,
} from "@/components/v2";
import { MissingExternalURLAttention } from "@/components/v2/Form";
import {
  extractWorkloadIdentityId,
  useActuatorV1Store,
  useDBGroupListByProject,
  useProjectByName,
  useWorkloadIdentityStore,
} from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import {
  type WorkloadIdentity,
  WorkloadIdentityConfig_ProviderType,
} from "@/types/proto-es/v1/workload_identity_service_pb";
import {
  getWorkloadIdentityProviderText,
  parseWorkloadIdentitySubjectPattern,
} from "@/utils";

const props = defineProps<{
  projectId: string;
}>();

const actuatorStore = useActuatorV1Store();
const workloadIdentityStore = useWorkloadIdentityStore();

const projectName = computed(() => `${projectNamePrefix}${props.projectId}`);
const { project } = useProjectByName(projectName);

const showCreateDrawer = ref(false);
const selectedIdentityName = ref<string | undefined>(undefined);
const selectedDatabaseNames = ref<string[]>([]);
const selectedDatabaseGroupName = ref<string | undefined>(undefined);
const targetTab = ref<"GROUP" | "DATABASE">("GROUP");
const activeTab = ref<WorkloadIdentityConfig_ProviderType | string>(
  WorkloadIdentityConfig_ProviderType.GITHUB
);
const useSelfhostRunner = ref(false);
const showSqlReviewYaml = ref(true);
const showReleaseYaml = ref(true);
const showGitlabCiYaml = ref(true);

const { dbGroupList } = useDBGroupListByProject(projectName);

const dbGroupOptions = computed(() => {
  return dbGroupList.value.map((group) => ({
    label: group.title || group.name,
    value: group.name,
  }));
});

const handleTargetTabChange = (tab: string) => {
  targetTab.value = tab as "GROUP" | "DATABASE";
  if (tab === "GROUP") {
    selectedDatabaseNames.value = [];
  } else {
    selectedDatabaseGroupName.value = undefined;
  }
};

const hasTargetSelected = computed(() => {
  return targetTab.value === "GROUP"
    ? !!selectedDatabaseGroupName.value
    : selectedDatabaseNames.value.length > 0;
});

const selectedIdentity = computed(() => {
  if (!selectedIdentityName.value) return undefined;
  return workloadIdentityStore.getWorkloadIdentity(selectedIdentityName.value);
});

const selectedConfig = computed(
  () => selectedIdentity.value?.workloadIdentityConfig
);

watch(
  () => selectedConfig.value,
  (config) => {
    activeTab.value =
      config?.providerType ?? WorkloadIdentityConfig_ProviderType.GITHUB;
  }
);

const parsedSubject = computed(() => {
  if (!selectedIdentity.value) {
    return;
  }
  return parseWorkloadIdentitySubjectPattern(selectedIdentity.value);
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

const bytebaseUrl = computed(() => {
  const url = actuatorStore.serverInfo?.externalUrl ?? "";
  return url.replace(/\/$/, "");
});

const workloadIdentityEmail = computed(() => {
  if (!selectedIdentityName.value) {
    return "{WORKLOAD_IDENTITY_EMAIL}";
  }
  return extractWorkloadIdentityId(selectedIdentityName.value);
});

const targetsString = computed(() => {
  if (targetTab.value === "GROUP" && selectedDatabaseGroupName.value) {
    return selectedDatabaseGroupName.value;
  }
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
        -d "{\\"token\\":\\"$GITLAB_OIDC_TOKEN\\",\\"email\\":\\"$BYTEBASE_WORKLOAD_IDENTITY\\"}" \\
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

const handleWorkloadIdentityCreated = (wi: WorkloadIdentity) => {
  selectedIdentityName.value = wi.name;
};
</script>
