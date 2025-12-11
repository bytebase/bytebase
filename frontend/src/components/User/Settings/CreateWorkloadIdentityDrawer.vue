<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="$t('settings.members.create-workload-identity')"
    >
      <template #default>
        <div class="flex flex-col gap-y-6">
          <!-- Email -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.email") }}
              <RequiredStar class="ml-0.5" />
            </label>
            <div class="flex items-center">
              <NInput
                v-model:value="state.emailPrefix"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                placeholder="github-deploy"
                :maxlength="100"
                class="flex-1"
              />
              <span class="ml-1 text-gray-500">@workload.bytebase.com</span>
            </div>
          </div>

          <!-- Name -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.name") }}
            </label>
            <NInput
              v-model:value="state.title"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="GitHub Deploy"
              :maxlength="200"
            />
          </div>

          <!-- Platform -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("settings.members.workload-identity-platform") }}
              <RequiredStar class="ml-0.5" />
            </label>
            <NSelect
              v-model:value="state.providerType"
              :options="platformOptions"
              @update:value="onPlatformChange"
            />
          </div>

          <!-- Owner -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("settings.members.workload-identity-owner") }}
              <RequiredStar class="ml-0.5" />
            </label>
            <NInput
              v-model:value="state.owner"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="my-org"
              :maxlength="200"
            />
          </div>

          <!-- Repository -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("settings.members.workload-identity-repo") }}
            </label>
            <NInput
              v-model:value="state.repo"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="my-repo"
              :maxlength="200"
            />
            <span class="text-xs text-gray-500">
              {{ $t("settings.members.workload-identity-repo-hint") }}
            </span>
          </div>

          <!-- Branch -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("settings.members.workload-identity-branch") }}
            </label>
            <NInput
              v-model:value="state.branch"
              :input-props="{ type: 'text', autocomplete: 'off' }"
              placeholder="main"
              :maxlength="200"
            />
            <span class="text-xs text-gray-500">
              {{ $t("settings.members.workload-identity-branch-hint") }}
            </span>
          </div>

          <!-- Advanced Settings -->
          <NCollapseTransition :show="showAdvanced">
            <div class="flex flex-col gap-y-4 mt-4 pt-4 border-t">
              <!-- Issuer URL -->
              <div class="flex flex-col gap-y-2">
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.members.workload-identity-issuer") }}
                </label>
                <NInput
                  v-model:value="state.issuerUrl"
                  :input-props="{ type: 'text', autocomplete: 'off' }"
                  :maxlength="500"
                />
              </div>

              <!-- Audience -->
              <div class="flex flex-col gap-y-2">
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.members.workload-identity-audience") }}
                </label>
                <NInput
                  v-model:value="state.audience"
                  :input-props="{ type: 'text', autocomplete: 'off' }"
                  :maxlength="500"
                />
              </div>

              <!-- Subject Pattern -->
              <div class="flex flex-col gap-y-2">
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.members.workload-identity-subject") }}
                </label>
                <NInput
                  v-model:value="state.subjectPattern"
                  :input-props="{ type: 'text', autocomplete: 'off' }"
                  :maxlength="500"
                />
              </div>
            </div>
          </NCollapseTransition>

          <NButton text @click="showAdvanced = !showAdvanced">
            {{ $t("settings.members.workload-identity-advanced") }}
            <template #icon>
              <heroicons-outline:chevron-down
                v-if="!showAdvanced"
                class="w-4 h-4"
              />
              <heroicons-outline:chevron-up v-else class="w-4 h-4" />
            </template>
          </NButton>

          <!-- Roles -->
          <div class="flex flex-col gap-y-2">
            <div>
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("settings.members.table.roles") }}
              </label>
            </div>
            <RoleSelect v-model:value="state.roles" :multiple="true" />
          </div>
        </div>
      </template>
      <template #footer>
        <div class="w-full flex flex-row items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton
            type="primary"
            :disabled="!allowConfirm"
            :loading="state.isRequesting"
            @click="tryCreateWorkloadIdentity"
          >
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NCollapseTransition, NInput, NSelect } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import { pushNotification, useUserStore, useWorkspaceV1Store } from "@/store";
import { PresetRoleType } from "@/types";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  ProviderType,
  UserSchema,
  UserType,
  WorkloadIdentityConfigSchema,
} from "@/types/proto-es/v1/user_service_pb";

interface LocalState {
  isRequesting: boolean;
  emailPrefix: string;
  title: string;
  providerType: ProviderType;
  owner: string;
  repo: string;
  branch: string;
  issuerUrl: string;
  audience: string;
  subjectPattern: string;
  roles: string[];
}

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", user: User): void;
}>();

const { t } = useI18n();
const userStore = useUserStore();
const workspaceStore = useWorkspaceV1Store();
const showAdvanced = ref(false);

const state = reactive<LocalState>({
  isRequesting: false,
  emailPrefix: "",
  title: "",
  providerType: ProviderType.PROVIDER_GITHUB,
  owner: "",
  repo: "",
  branch: "",
  issuerUrl: "",
  audience: "",
  subjectPattern: "",
  roles: [PresetRoleType.WORKSPACE_MEMBER],
});

const platformOptions = [
  { label: "GitHub Actions", value: ProviderType.PROVIDER_GITHUB },
  { label: "GitLab CI", value: ProviderType.PROVIDER_GITLAB },
  { label: "Bitbucket Pipelines", value: ProviderType.PROVIDER_BITBUCKET },
  { label: "Azure DevOps", value: ProviderType.PROVIDER_AZURE_DEVOPS },
];

const platformPresets: Record<
  ProviderType,
  { issuerUrl: string; audience: string }
> = {
  [ProviderType.PROVIDER_GITHUB]: {
    issuerUrl: "https://token.actions.githubusercontent.com",
    audience: "",
  },
  [ProviderType.PROVIDER_GITLAB]: {
    issuerUrl: "https://gitlab.com",
    audience: "https://gitlab.com",
  },
  [ProviderType.PROVIDER_BITBUCKET]: {
    issuerUrl: "",
    audience: "",
  },
  [ProviderType.PROVIDER_AZURE_DEVOPS]: {
    issuerUrl: "",
    audience: "api://AzureADTokenExchange",
  },
  [ProviderType.PROVIDER_TYPE_UNSPECIFIED]: {
    issuerUrl: "",
    audience: "",
  },
};

const onPlatformChange = (value: ProviderType) => {
  const preset = platformPresets[value];
  if (preset) {
    state.issuerUrl = preset.issuerUrl;
    state.audience = preset.audience;
  }
};

// Auto-build subject pattern based on platform and inputs
const computedSubjectPattern = computed(() => {
  const { providerType, owner, repo, branch } = state;

  switch (providerType) {
    case ProviderType.PROVIDER_GITHUB:
      if (!repo) {
        return `repo:${owner}/*`;
      }
      if (!branch) {
        return `repo:${owner}/${repo}:*`;
      }
      return `repo:${owner}/${repo}:ref:refs/heads/${branch}`;
    case ProviderType.PROVIDER_GITLAB:
      if (!repo) {
        return `project_path:${owner}/*`;
      }
      if (!branch) {
        return `project_path:${owner}/${repo}:*`;
      }
      return `project_path:${owner}/${repo}:ref_type:branch:ref:${branch}`;
    default:
      return "";
  }
});

// Watch for owner/repo/branch changes and update subject pattern
watch(
  () => [state.owner, state.repo, state.branch, state.providerType],
  () => {
    if (!showAdvanced.value) {
      state.subjectPattern = computedSubjectPattern.value;
    }
  },
  { immediate: true }
);

// Initialize defaults
onPlatformChange(state.providerType);

const allowConfirm = computed(() => {
  if (!state.emailPrefix) {
    return false;
  }
  if (!state.owner) {
    return false;
  }
  if (!state.issuerUrl) {
    return false;
  }
  return true;
});

const tryCreateWorkloadIdentity = async () => {
  state.isRequesting = true;
  try {
    const email = `${state.emailPrefix}@workload.bytebase.com`;
    const createdUser = await userStore.createUser(
      create(UserSchema, {
        name: "",
        email,
        title: state.title || state.emailPrefix,
        userType: UserType.WORKLOAD_IDENTITY,
        password: "",
        phone: "",
        mfaEnabled: false,
        workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
          providerType: state.providerType,
          issuerUrl: state.issuerUrl,
          allowedAudiences: state.audience ? [state.audience] : [],
          subjectPattern: state.subjectPattern,
        }),
      })
    );

    if (state.roles.length > 0) {
      await workspaceStore.patchIamPolicy([
        {
          member: `workloadIdentity:${createdUser.email}`,
          roles: state.roles,
        },
      ]);
    }

    emit("created", createdUser);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.created"),
    });
    emit("close");
  } finally {
    state.isRequesting = false;
  }
};
</script>
