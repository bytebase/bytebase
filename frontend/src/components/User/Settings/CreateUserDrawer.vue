<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="
        isEditMode
          ? $t('settings.members.update-user')
          : $t('settings.members.add-user')
      "
    >
      <template #default>
        <div class="flex flex-col gap-y-6">
          <div class="w-full flex flex-col gap-y-2">
            <div class="flex items-center gap-x-1">
              <div class="text-sm font-medium">{{ $t("common.type") }}</div>
              <a
                href="https://docs.bytebase.com/get-started/terraform?source=console"
                target="_blank"
              >
                <heroicons-outline:question-mark-circle class="w-4 h-4" />
              </a>
            </div>
            <NRadioGroup
              v-model:value="state.user.userType"
              :disabled="isEditMode"
            >
              <NRadio :value="UserType.USER" :label="$t('common.user')" />
              <NRadio
                :value="UserType.SERVICE_ACCOUNT"
                :label="$t('settings.members.service-account')"
              />
              <NRadio
                :value="UserType.WORKLOAD_IDENTITY"
                :label="$t('settings.members.workload-identity')"
              />
            </NRadioGroup>
          </div>

          <!-- Workload Identity Fields -->
          <template v-if="state.user.userType === UserType.WORKLOAD_IDENTITY">
            <!-- Name -->
            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("common.name") }}
              </label>
              <NInput
                v-model:value="state.user.title"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                placeholder="GitHub Deploy"
                :maxlength="200"
                :disabled="!allowUpdate"
              />
            </div>

            <!-- Email -->
            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("common.email") }}
                <RequiredStar class="ml-0.5" />
              </label>
              <EmailInput
                v-model:value="state.user.email"
                :domain="workloadIdentitySuffix"
                :show-domain="true"
                :disabled="isEditMode"
              />
            </div>

            <!-- Platform -->
            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("settings.members.workload-identity-platform") }}
                <RequiredStar class="ml-0.5" />
              </label>
              <NSelect
                v-model:value="state.wif.providerType"
                :options="platformOptions"
                :disabled="!allowUpdate"
                @update:value="onPlatformChange"
              />
            </div>

            <!-- Owner / Group -->
            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                <template
                  v-if="
                    state.wif.providerType ===
                    WorkloadIdentityConfig_ProviderType.GITLAB
                  "
                >
                  {{ $t("settings.members.workload-identity-group") }}
                </template>
                <template v-else>
                  {{ $t("settings.members.workload-identity-owner") }}
                </template>
                <RequiredStar class="ml-0.5" />
              </label>
              <NInput
                v-model:value="state.wif.owner"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                :placeholder="
                  state.wif.providerType ===
                  WorkloadIdentityConfig_ProviderType.GITLAB
                    ? 'my-group'
                    : 'my-org'
                "
                :maxlength="200"
                :disabled="!allowUpdate"
              />
            </div>

            <!-- Repository / Project -->
            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                <template
                  v-if="
                    state.wif.providerType ===
                    WorkloadIdentityConfig_ProviderType.GITLAB
                  "
                >
                  {{ $t("settings.members.workload-identity-project") }}
                </template>
                <template v-else>
                  {{ $t("settings.members.workload-identity-repo") }}
                </template>
              </label>
              <NInput
                v-model:value="state.wif.repo"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                :placeholder="
                  state.wif.providerType ===
                  WorkloadIdentityConfig_ProviderType.GITLAB
                    ? 'my-project'
                    : 'my-repo'
                "
                :maxlength="200"
                :disabled="!allowUpdate"
              />
              <span class="text-xs text-gray-500">
                <template
                  v-if="
                    state.wif.providerType ===
                    WorkloadIdentityConfig_ProviderType.GITLAB
                  "
                >
                  {{
                    $t("settings.members.workload-identity-project-hint")
                  }}
                </template>
                <template v-else>
                  {{ $t("settings.members.workload-identity-repo-hint") }}
                </template>
              </span>
            </div>

            <!-- Allowed Branches/Tags (GitLab only) -->
            <div
              v-if="
                state.wif.providerType ===
                WorkloadIdentityConfig_ProviderType.GITLAB
              "
              class="flex flex-col gap-y-2"
            >
              <label class="block text-sm font-medium leading-5 text-control">
                {{
                  $t(
                    "settings.members.workload-identity-allowed-branches-tags"
                  )
                }}
              </label>
              <NSelect
                v-model:value="state.wif.refType"
                :options="refTypeOptions"
                :disabled="!allowUpdate"
              />
            </div>

            <!-- Branch (GitHub) or Branch/Tag (GitLab when specific is selected) -->
            <div
              v-if="
                state.wif.providerType ===
                  WorkloadIdentityConfig_ProviderType.GITHUB ||
                state.wif.refType !== 'all'
              "
              class="flex flex-col gap-y-2"
            >
              <label class="block text-sm font-medium leading-5 text-control">
                <template
                  v-if="
                    state.wif.providerType ===
                      WorkloadIdentityConfig_ProviderType.GITLAB &&
                    state.wif.refType === 'tag'
                  "
                >
                  {{ $t("settings.members.workload-identity-tag") }}
                </template>
                <template v-else>
                  {{ $t("settings.members.workload-identity-branch") }}
                </template>
              </label>
              <NInput
                v-model:value="state.wif.branch"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                :placeholder="state.wif.refType === 'tag' ? 'v1.0.0' : 'main'"
                :maxlength="200"
                :disabled="!allowUpdate"
              />
              <span class="text-xs text-gray-500">
                <template
                  v-if="
                    state.wif.providerType ===
                      WorkloadIdentityConfig_ProviderType.GITLAB &&
                    state.wif.refType === 'tag'
                  "
                >
                  {{ $t("settings.members.workload-identity-tag-hint") }}
                </template>
                <template v-else>
                  {{ $t("settings.members.workload-identity-branch-hint") }}
                </template>
              </span>
            </div>

            <!-- Advanced Settings -->
            <NCollapseTransition :show="state.wif.showAdvanced">
              <div class="flex flex-col gap-y-6 pt-6 border-t">
                <!-- Issuer URL / GitLab URL -->
                <div class="flex flex-col gap-y-2">
                  <label
                    class="block text-sm font-medium leading-5 text-control"
                  >
                    <template
                      v-if="
                        state.wif.providerType ===
                        WorkloadIdentityConfig_ProviderType.GITLAB
                      "
                    >
                      {{
                        $t("settings.members.workload-identity-gitlab-url")
                      }}
                    </template>
                    <template v-else>
                      {{ $t("settings.members.workload-identity-issuer") }}
                    </template>
                  </label>
                  <NInput
                    v-model:value="state.wif.issuerUrl"
                    :input-props="{ type: 'text', autocomplete: 'off' }"
                    :maxlength="500"
                    :disabled="!allowUpdate"
                  />
                  <span
                    v-if="
                      state.wif.providerType ===
                      WorkloadIdentityConfig_ProviderType.GITLAB
                    "
                    class="text-xs text-gray-500"
                  >
                    {{
                      $t(
                        "settings.members.workload-identity-gitlab-url-hint"
                      )
                    }}
                  </span>
                </div>

                <!-- Audience -->
                <div class="flex flex-col gap-y-2">
                  <label
                    class="block text-sm font-medium leading-5 text-control"
                  >
                    {{ $t("settings.members.workload-identity-audience") }}
                  </label>
                  <NInput
                    v-model:value="state.wif.audience"
                    :input-props="{ type: 'text', autocomplete: 'off' }"
                    :maxlength="500"
                    :disabled="!allowUpdate"
                  />
                </div>

                <!-- Subject Pattern -->
                <div class="flex flex-col gap-y-2">
                  <label
                    class="block text-sm font-medium leading-5 text-control"
                  >
                    {{ $t("settings.members.workload-identity-subject") }}
                  </label>
                  <NInput
                    v-model:value="state.wif.subjectPattern"
                    :input-props="{ type: 'text', autocomplete: 'off' }"
                    :maxlength="500"
                    :disabled="!allowUpdate"
                  />
                </div>
              </div>
            </NCollapseTransition>

            <NButton
              text
              @click="state.wif.showAdvanced = !state.wif.showAdvanced"
            >
              {{ $t("settings.members.workload-identity-advanced") }}
              <template #icon>
                <heroicons-outline:chevron-down
                  v-if="!state.wif.showAdvanced"
                  class="w-4 h-4"
                />
                <heroicons-outline:chevron-up v-else class="w-4 h-4" />
              </template>
            </NButton>
          </template>

          <!-- Regular User / Service Account Fields -->
          <template v-else>
            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("common.name") }}
              </label>
              <NInput
                v-model:value="state.user.title"
                :input-props="{ type: 'text', autocomplete: 'off' }"
                placeholder="Foo"
                :maxlength="200"
                :disabled="!allowUpdate"
              />
            </div>

            <div class="flex flex-col gap-y-2">
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("common.email") }}
                <RequiredStar class="ml-0.5" />
              </label>
              <EmailInput
                v-model:value="state.user.email"
                :domain="
                  state.user.userType === UserType.SERVICE_ACCOUNT
                   ? serviceAccountSuffix
                   : ''
                "
                :show-domain="state.user.userType === UserType.SERVICE_ACCOUNT"
                :disabled="isEditMode"
              />
            </div>
          </template>

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="[
              'bb.workspaces.setIamPolicy'
            ]"
          >
            <div class="flex flex-col gap-y-2">
              <div>
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.members.table.roles") }}
                </label>
              </div>
              <RoleSelect
                  v-model:value="state.roles"
                  :multiple="true"
                  :disabled="slotProps.disabled"
                />
            </div>
          </PermissionGuardWrapper>

          <template v-if="state.user.userType === UserType.USER">
            <div class="flex flex-col gap-y-2">
              <div>
                <label class="block text-sm font-medium leading-5 text-control">
                  {{ $t("settings.profile.phone") }}
                </label>
                <span class="textinfolabel text-sm">
                  {{ $t("settings.profile.phone-tips") }}
                </span>
              </div>
              <NInput
                v-model:value="state.user.phone"
                type="text"
                :input-props="{
                  type: 'tel',
                  autocomplete: 'new-password',
                }"
                :disabled="!allowUpdate"
              />
            </div>

            <UserPassword
              ref="userPasswordRef"
              v-model:password="state.user.password"
              v-model:password-confirm="state.passwordConfirm"
              :password-restriction="passwordRestrictionSetting"
              :disabled="!allowUpdate"
            />
          </template>
        </div>
      </template>
      <template #footer>
        <div class="w-full flex flex-row items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">
            {{ $t("common.cancel") }}
          </NButton>

          <PermissionGuardWrapper
            v-slot="slotProps"
            :permissions="[
              isEditMode ? updatePermission : createPermission
            ]"
          >
            <NButton
              type="primary"
              :disabled="!allowConfirm || slotProps.disabled"
              :loading="state.isRequesting"
              @click="createOrUpdateUser"
            >
              {{
                isEditMode ? $t("common.update") : $t("common.confirm")
              }}
            </NButton>
          </PermissionGuardWrapper>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import {
  NButton,
  NCollapseTransition,
  NInput,
  NRadio,
  NRadioGroup,
  NSelect,
} from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import PermissionGuardWrapper from "@/components/Permission/PermissionGuardWrapper.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import {
  ensureServiceAccountFullName,
  ensureWorkloadIdentityFullName,
  getUserFullNameByType,
  pushNotification,
  serviceAccountToUser,
  useServiceAccountStore,
  useSettingV1Store,
  useUserStore,
  useWorkloadIdentityStore,
  useWorkspaceV1Store,
  workloadIdentityToUser,
} from "@/store";
import {
  getServiceAccountNameInBinding,
  getUserEmailInBinding,
  getWorkloadIdentityNameInBinding,
  serviceAccountSuffix,
  UNKNOWN_USER_NAME,
  unknownUser,
  workloadIdentitySuffix,
} from "@/types";
import { PresetRoleType } from "@/types/iam";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UpdateUserRequestSchema,
  UserSchema,
  UserType,
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";
import UserPassword from "./UserPassword.vue";

interface WifState {
  emailPrefix: string;
  providerType: WorkloadIdentityConfig_ProviderType;
  owner: string;
  repo: string;
  branch: string;
  refType: "branch" | "tag" | "all";
  issuerUrl: string;
  audience: string;
  subjectPattern: string;
  showAdvanced: boolean;
}

interface LocalState {
  isRequesting: boolean;
  user: User;
  roles: string[];
  passwordConfirm: string;
  wif: WifState;
}

const props = defineProps<{
  user?: User;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", user: User): void;
  (event: "updated", user: User): void;
}>();

const workspaceStore = useWorkspaceV1Store();

const { t } = useI18n();
const userStore = useUserStore();
const serviceAccountStore = useServiceAccountStore();
const workloadIdentityStore = useWorkloadIdentityStore();
const userPasswordRef = ref<InstanceType<typeof UserPassword>>();

const state = reactive<LocalState>({
  isRequesting: false,
  user: unknownUser(),
  roles: [PresetRoleType.WORKSPACE_MEMBER],
  passwordConfirm: "",
  wif: {
    emailPrefix: "",
    providerType: WorkloadIdentityConfig_ProviderType.GITHUB,
    owner: "",
    repo: "",
    branch: "",
    refType: "all" as "branch" | "tag" | "all",
    issuerUrl: "https://token.actions.githubusercontent.com",
    audience: "",
    subjectPattern: "",
    showAdvanced: false,
  },
});

const isEditMode = computed(
  () => !!props.user && props.user.name !== unknownUser().name
);

const updatePermission = computed(() => {
  switch (state.user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return "bb.serviceAccounts.update";
    case UserType.WORKLOAD_IDENTITY:
      return "bb.workloadIdentities.update";
    default:
      return "bb.users.update";
  }
});

const createPermission = computed(() => {
  switch (state.user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return "bb.serviceAccounts.create";
    case UserType.WORKLOAD_IDENTITY:
      return "bb.workloadIdentities.create";
    default:
      return "bb.users.create";
  }
});

const allowUpdate = computed(() => {
  if (!isEditMode.value) {
    return true;
  }
  return hasWorkspacePermissionV2(updatePermission.value);
});

const initialRoles = computed(() => {
  if (!props.user || props.user.name === UNKNOWN_USER_NAME) {
    return [PresetRoleType.WORKSPACE_MEMBER];
  }

  const roles = workspaceStore.userMapToRoles.get(
    getUserFullNameByType(props.user)
  );
  return roles ? [...roles] : [];
});

// Parse subject pattern and extract owner/repo/branch/refType
const parseSubjectPattern = (pattern: string) => {
  const { providerType } = state.wif;

  if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
    // GitHub patterns:
    // repo:owner/*
    // repo:owner/repo:*
    // repo:owner/repo:ref:refs/heads/branch
    const match = pattern.match(/^repo:([^/]+)\/(.*)$/);
    if (match) {
      const owner = match[1];
      const rest = match[2];
      if (rest === "*") {
        return { owner, repo: "", branch: "" };
      }
      const repoMatch = rest.match(/^([^:]+):(.*)$/);
      if (repoMatch) {
        const repo = repoMatch[1];
        const refPart = repoMatch[2];
        if (refPart === "*") {
          return { owner, repo, branch: "" };
        }
        const branchMatch = refPart.match(/^ref:refs\/heads\/(.+)$/);
        if (branchMatch) {
          return { owner, repo, branch: branchMatch[1] };
        }
      }
    }
  }

  if (providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    // GitLab patterns:
    // project_path:group/*
    // project_path:group/project:*
    // project_path:group/project:ref_type:branch:ref:main
    // project_path:group/project:ref_type:tag:ref:v1.0.0
    const match = pattern.match(/^project_path:([^/]+)\/(.*)$/);
    if (match) {
      const owner = match[1];
      const rest = match[2];
      if (rest === "*") {
        return { owner, repo: "", branch: "", refType: "all" as const };
      }
      const projectMatch = rest.match(/^([^:]+):(.*)$/);
      if (projectMatch) {
        const repo = projectMatch[1];
        const refPart = projectMatch[2];
        if (refPart === "*") {
          return { owner, repo, branch: "", refType: "all" as const };
        }
        const refTypeMatch = refPart.match(/^ref_type:(branch|tag):ref:(.+)$/);
        if (refTypeMatch) {
          return {
            owner,
            repo,
            branch: refTypeMatch[2],
            refType: refTypeMatch[1] as "branch" | "tag",
          };
        }
      }
    }
  }

  return null;
};

watch(
  () => props.user,
  (user) => {
    if (!user) {
      return;
    }
    state.user = cloneDeep(create(UserSchema, user));
    state.roles = [...initialRoles.value];

    if (user.userType === UserType.WORKLOAD_IDENTITY) {
      const config = user.workloadIdentityConfig;
      if (config) {
        state.wif.emailPrefix = user.email.split("@")[0];
        state.wif.providerType = config.providerType;
        state.wif.issuerUrl = config.issuerUrl;
        state.wif.audience = config.allowedAudiences[0] || "";
        state.wif.subjectPattern = config.subjectPattern;

        const parsed = parseSubjectPattern(config.subjectPattern);
        if (parsed) {
          state.wif.owner = parsed.owner;
          state.wif.repo = parsed.repo;
          state.wif.branch = parsed.branch;
          if ("refType" in parsed && parsed.refType) {
            state.wif.refType = parsed.refType;
          }
        }
      }
    }
  },
  {
    immediate: true,
  }
);

const platformOptions = [
  {
    label: "GitHub Actions",
    value: WorkloadIdentityConfig_ProviderType.GITHUB,
  },
  {
    label: "GitLab CI",
    value: WorkloadIdentityConfig_ProviderType.GITLAB,
  },
];

const platformPresets: Partial<
  Record<
    WorkloadIdentityConfig_ProviderType,
    { issuerUrl: string; audience: string }
  >
> = {
  [WorkloadIdentityConfig_ProviderType.GITHUB]: {
    issuerUrl: "https://token.actions.githubusercontent.com",
    audience: "",
  },
  [WorkloadIdentityConfig_ProviderType.GITLAB]: {
    issuerUrl: "https://gitlab.com",
    audience: "",
  },
};

const refTypeOptions = computed(() => {
  if (state.wif.providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    return [
      {
        label: t("settings.members.workload-identity-all-branches-tags"),
        value: "all",
      },
      {
        label: t("settings.members.workload-identity-specific-branch"),
        value: "branch",
      },
      {
        label: t("settings.members.workload-identity-specific-tag"),
        value: "tag",
      },
    ];
  }
  return [];
});

const onPlatformChange = (value: WorkloadIdentityConfig_ProviderType) => {
  const preset = platformPresets[value];
  if (preset) {
    state.wif.issuerUrl = preset.issuerUrl;
    state.wif.audience = preset.audience;
  }
  state.wif.refType = "all";
  state.wif.branch = "";
};

// Auto-build subject pattern based on platform and inputs
const computedSubjectPattern = computed(() => {
  const { owner, repo, branch, providerType, refType } = state.wif;

  if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
    if (!repo) {
      return `repo:${owner}/*`;
    }
    if (!branch) {
      return `repo:${owner}/${repo}:*`;
    }
    return `repo:${owner}/${repo}:ref:refs/heads/${branch}`;
  }

  if (providerType === WorkloadIdentityConfig_ProviderType.GITLAB) {
    if (!repo) {
      return `project_path:${owner}/*`;
    }
    if (refType === "all" || !branch) {
      return `project_path:${owner}/${repo}:*`;
    }
    return `project_path:${owner}/${repo}:ref_type:${refType}:ref:${branch}`;
  }

  return "";
});

// Flag to prevent circular updates
let isUpdatingFromPattern = false;
let isUpdatingFromFields = false;

// Watch for owner/repo/branch changes and update subject pattern
watch(
  () => [
    state.wif.owner,
    state.wif.repo,
    state.wif.branch,
    state.wif.providerType,
    state.wif.refType,
  ],
  () => {
    if (isUpdatingFromPattern) return;
    isUpdatingFromFields = true;
    state.wif.subjectPattern = computedSubjectPattern.value;
    isUpdatingFromFields = false;
  },
  { immediate: true }
);

// Watch for subject pattern changes and update fields (reverse binding)
watch(
  () => state.wif.subjectPattern,
  (newPattern) => {
    if (isUpdatingFromFields) return;
    const parsed = parseSubjectPattern(newPattern);
    if (parsed) {
      isUpdatingFromPattern = true;
      state.wif.owner = parsed.owner;
      state.wif.repo = parsed.repo;
      state.wif.branch = parsed.branch;
      if ("refType" in parsed && parsed.refType) {
        state.wif.refType = parsed.refType;
      }
      isUpdatingFromPattern = false;
    }
  }
);

const passwordRestrictionSetting = computed(
  () => useSettingV1Store().workspaceProfile.passwordRestriction
);

const allowConfirm = computed(() => {
  if (state.user.userType === UserType.WORKLOAD_IDENTITY) {
    if (!state.wif.emailPrefix) {
      return false;
    }
    if (!state.wif.owner) {
      return false;
    }
    if (!state.wif.issuerUrl) {
      return false;
    }
    return true;
  }

  if (!state.user.email) {
    return false;
  }
  if (userPasswordRef.value?.passwordHint) {
    return false;
  }
  if (userPasswordRef.value?.passwordMismatch) {
    return false;
  }

  return true;
});

const extractUserTitle = (email: string): string => {
  const atIndex = email.indexOf("@");
  if (atIndex !== -1) {
    return email.substring(0, atIndex);
  }
  // If there is no @, we just return the email as title.
  return email;
};

const createOrUpdateUser = async () => {
  state.isRequesting = true;
  try {
    if (isEditMode.value) {
      await updateUser();
    } else {
      await createUser();
    }
  } catch {
    // nothing
  } finally {
    state.isRequesting = false;
  }
};

const convertUserToMember = (user: User) => {
  switch (user.userType) {
    case UserType.SERVICE_ACCOUNT:
      return getServiceAccountNameInBinding(user.email);
    case UserType.WORKLOAD_IDENTITY:
      return getWorkloadIdentityNameInBinding(user.email);
    default:
      return getUserEmailInBinding(user.email);
  }
};

const createUser = async () => {
  let createdUser: User;

  switch (state.user.userType) {
    case UserType.WORKLOAD_IDENTITY: {
      const wi = await workloadIdentityStore.createWorkloadIdentity(
        state.wif.emailPrefix,
        {
          title: state.user.title || state.wif.emailPrefix,
          workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
            providerType: state.wif.providerType,
            issuerUrl: state.wif.issuerUrl,
            allowedAudiences: state.wif.audience ? [state.wif.audience] : [],
            subjectPattern: state.wif.subjectPattern,
          }),
        }
      );
      createdUser = workloadIdentityToUser(wi);
      break;
    }
    case UserType.SERVICE_ACCOUNT: {
      const serviceAccountId = state.user.email.split("@")[0];
      const sa = await serviceAccountStore.createServiceAccount(
        serviceAccountId,
        {
          title: state.user.title,
        }
      );
      createdUser = serviceAccountToUser(sa);
      break;
    }
    default: {
      createdUser = await userStore.createUser({
        ...state.user,
        title: state.user.title || extractUserTitle(state.user.email),
        password: state.user.password,
      });
    }
  }

  if (state.roles.length > 0) {
    await workspaceStore.patchIamPolicy([
      {
        member: convertUserToMember(createdUser),
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
};

const updateUser = async () => {
  const user = props.user;
  if (!user) {
    return;
  }

  const updateMask: string[] = [];
  const payload = create(UserSchema, state.user);
  if (payload.title !== user.title) {
    updateMask.push("title");
  }

  let updatedUser: User = user;

  switch (payload.userType) {
    case UserType.WORKLOAD_IDENTITY: {
      const wi = await workloadIdentityStore.updateWorkloadIdentity(
        {
          name: ensureWorkloadIdentityFullName(payload.email),
          title: payload.title,
          workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
            providerType: state.wif.providerType,
            issuerUrl: state.wif.issuerUrl,
            allowedAudiences: state.wif.audience ? [state.wif.audience] : [],
            subjectPattern: state.wif.subjectPattern,
          }),
        },
        create(FieldMaskSchema, {
          paths: [...updateMask, "workload_identity_config"],
        })
      );
      updatedUser = workloadIdentityToUser(wi);
      break;
    }
    case UserType.SERVICE_ACCOUNT: {
      if (updateMask.length > 0) {
        const sa = await serviceAccountStore.updateServiceAccount(
          {
            name: ensureServiceAccountFullName(payload.email),
            title: payload.title,
          },
          create(FieldMaskSchema, {
            paths: [...updateMask],
          })
        );
        updatedUser = serviceAccountToUser(sa);
      }
      break;
    }
    default: {
      if (payload.phone !== user.phone) {
        updateMask.push("phone");
      }
      if (payload.password) {
        updateMask.push("password");
      }

      if (updateMask.length > 0) {
        updatedUser = await userStore.updateUser(
          create(UpdateUserRequestSchema, {
            user: payload,
            updateMask: create(FieldMaskSchema, {
              paths: updateMask,
            }),
          })
        );
      }
    }
  }

  if (!isEqual([...initialRoles.value].sort(), [...state.roles].sort())) {
    await workspaceStore.patchIamPolicy([
      {
        member: convertUserToMember(updatedUser),
        roles: state.roles,
      },
    ]);
  }

  emit("updated", updatedUser);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });
  emit("close");
};
</script>
