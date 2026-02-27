<template>
  <div class="w-full px-4 flex flex-col gap-y-6 py-4">
    <!-- Header -->
    <div>
      <h2 class="text-lg font-medium">
        {{ $t("gitops.overview.title") }}
      </h2>
      <p class="textinfolabel mt-1">
        {{ $t("gitops.overview.description") }}
      </p>
    </div>

    <!-- Warning if external URL not configured -->
    <MissingExternalURLAttention />

    <!-- Section 1: Workload Identity -->
    <div class="flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <h3 class="text-base font-medium">
          {{ $t("gitops.workload-identity.title") }}
        </h3>
        <p class="text-sm text-control-light">
          {{ $t("gitops.workload-identity.description") }}
        </p>
      </div>
      <div class="flex items-center gap-x-3">
        <NSelect
          v-model:value="selectedIdentityEmail"
          :options="identityOptions"
          :placeholder="$t('gitops.workload-identity.select-placeholder')"
          clearable
          class="max-w-md"
        />
        <NButton @click="showCreateDrawer = true">
          {{ $t("common.create") }}
        </NButton>
      </div>
      <p
        v-if="identityOptions.length === 0 && !isLoading"
        class="text-sm text-control-light"
      >
        {{ $t("gitops.workload-identity.no-identity") }}
      </p>
      <!-- Repository link derived from workload identity -->
      <a
        v-if="repoUrl"
        :href="repoUrl"
        target="_blank"
        class="text-sm text-accent hover:underline"
      >
        {{ repoUrl }} &rarr;
      </a>
    </div>

    <!-- Section 2: CI/CD Workflow YAML -->
    <div class="flex flex-col gap-y-3">
      <div class="flex flex-col gap-y-1">
        <div class="flex items-center gap-x-2">
          <h3 class="text-base font-medium">
            {{ $t("gitops.workflow.title") }}
          </h3>
          <CopyButton :content="activeTabContent" />
        </div>
        <p class="text-sm text-control-light">
          {{ $t("gitops.workflow.description") }}
        </p>
      </div>
      <NTabs v-model:value="activeTab" type="line" animated>
        <NTabPane
          v-for="tab in tabs"
          :key="tab.id"
          :name="tab.id"
          :tab="tab.title"
        >
          <p class="text-sm text-control-light mt-3">
            {{ $t("gitops.workflow.file-hint", { filePath: tab.filePath }) }}
          </p>
          <NInput
            :value="tab.content"
            type="textarea"
            readonly
            :autosize="{ minRows: 10, maxRows: 30 }"
            class="font-mono text-sm mt-2"
          />
        </NTabPane>
      </NTabs>
    </div>

    <!-- Section 3: Documentation link -->
    <div>
      <a
        href="https://www.bytebase.com/docs/vcs-integration/overview/"
        target="_blank"
        class="text-accent hover:underline"
      >
        {{ $t("gitops.documentation") }} &rarr;
      </a>
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
import { NButton, NInput, NSelect, NTabPane, NTabs } from "naive-ui";
import { computed, onMounted, ref, watch } from "vue";
import CreateWorkloadIdentityDrawer from "@/components/User/Settings/CreateWorkloadIdentityDrawer.vue";
import { CopyButton, MissingExternalURLAttention } from "@/components/v2";
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
const { project: _project } = useProjectByName(projectName);

const showCreateDrawer = ref(false);
const selectedIdentityEmail = ref<string | null>(null);
const isLoading = ref(false);
const identityOptions = ref<{ label: string; value: string }[]>([]);
const identityMap = ref<Map<string, WorkloadIdentity>>(new Map());
const activeTab = ref("github-actions");

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
  if (!url) {
    return "{BYTEBASE_URL}";
  }
  return url.replace(/\/$/, "");
});

const serviceAccountEmail = computed(() => {
  if (!selectedIdentityEmail.value) {
    return "{WORKLOAD_IDENTITY_EMAIL}";
  }
  return selectedIdentityEmail.value;
});

const githubActionsYaml = computed(() => {
  return `name: Bytebase Schema Migration
on:
  push:
    branches: ["${branch.value}"]

jobs:
  bytebase:
    runs-on: ubuntu-latest
    container:
      image: bytebase/bytebase-action
    steps:
      - uses: actions/checkout@v4
      - name: Bytebase Push
        env:
          BYTEBASE_URL: ${bytebaseUrl.value}
          BYTEBASE_SERVICE_ACCOUNT: ${serviceAccountEmail.value}
          BYTEBASE_SERVICE_ACCOUNT_SECRET: \${{ secrets.BYTEBASE_SERVICE_ACCOUNT_SECRET }}
        run: |
          bytebase-action check \\
            --file-pattern "migrations/*.sql" \\
            --project "projects/${props.projectId}" \\
            --targets "instances/{instance}/databases/{database}"
          bytebase-action rollout \\
            --file-pattern "migrations/*.sql" \\
            --project "projects/${props.projectId}" \\
            --targets "instances/{instance}/databases/{database}"`;
});

const gitlabCiYaml = computed(() => {
  return `stages:
  - bytebase

bytebase:
  stage: bytebase
  image: bytebase/bytebase-action
  variables:
    BYTEBASE_URL: ${bytebaseUrl.value}
    BYTEBASE_SERVICE_ACCOUNT: ${serviceAccountEmail.value}
  script:
    - |
      bytebase-action check \\
        --file-pattern "migrations/*.sql" \\
        --project "projects/${props.projectId}" \\
        --targets "instances/{instance}/databases/{database}"
      bytebase-action rollout \\
        --file-pattern "migrations/*.sql" \\
        --project "projects/${props.projectId}" \\
        --targets "instances/{instance}/databases/{database}"
  rules:
    - if: $CI_COMMIT_BRANCH == "${branch.value}"`;
});

const tabs = computed(() => [
  {
    id: "github-actions",
    title: "GitHub Actions",
    content: githubActionsYaml.value,
    filePath: ".github/workflows/bytebase.yml",
  },
  {
    id: "gitlab-ci",
    title: "GitLab CI",
    content: gitlabCiYaml.value,
    filePath: ".gitlab-ci.yml",
  },
]);

const activeTabContent = computed(() => {
  const tab = tabs.value.find((t) => t.id === activeTab.value);
  return tab?.content ?? "";
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
