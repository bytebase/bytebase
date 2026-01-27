<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="
        isEditMode
          ? $t('settings.members.update-workload-identity')
          : $t('settings.members.add-workload-identity')
      "
    >
      <template #default>
        <div class="flex flex-col gap-y-6">
          <!-- Name -->
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.name") }}
            </label>
            <NInput
              v-model:value="state.workloadIdentity.title"
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
              v-model:value="state.workloadIdentity.email"
              :domain="emailSuffix"
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

          <div class="flex flex-col gap-y-2">
            <div>
              <label class="block text-sm font-medium leading-5 text-control">
                {{ $t("settings.members.table.roles") }}
              </label>
            </div>
            <RoleSelect
              v-model:value="state.roles"
              :multiple="true"
              :disabled="!allowUpdateRoles"
              :project="project"
            />
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
            @click="createOrUpdateWorkloadIdentity"
          >
            {{
              isEditMode ? $t("common.update") : $t("common.confirm")
            }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { FieldMaskSchema } from "@bufbuild/protobuf/wkt";
import { cloneDeep, isEqual } from "lodash-es";
import { NButton, NCollapseTransition, NInput, NSelect } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import {
  ensureWorkloadIdentityFullName,
  pushNotification,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useWorkloadIdentityStore,
  useWorkspaceV1Store,
  workloadIdentityToUser,
} from "@/store";
import {
  getWorkloadIdentityNameInBinding,
  getWorkloadIdentitySuffix,
  UNKNOWN_USER_NAME,
  unknownUser,
} from "@/types";
import { PresetRoleType } from "@/types/iam";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import {
  UserSchema,
  WorkloadIdentityConfig_ProviderType,
  WorkloadIdentityConfigSchema,
} from "@/types/proto-es/v1/user_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

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
  workloadIdentity: User;
  roles: string[];
  wif: WifState;
}

const props = defineProps<{
  workloadIdentity?: User;
  projectId?: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", user: User): void;
  (event: "updated", user: User): void;
}>();

const { t } = useI18n();
const workloadIdentityStore = useWorkloadIdentityStore();
const workspaceStore = useWorkspaceV1Store();
const projectStore = useProjectV1Store();
const projectIamPolicyStore = useProjectIamPolicyStore();

const state = reactive<LocalState>({
  isRequesting: false,
  workloadIdentity: unknownUser(),
  roles: props.projectId ? [] : [PresetRoleType.WORKSPACE_MEMBER],
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

const project = computed(() => {
  if (!props.projectId) return undefined;
  return projectStore.getProjectByName(`projects/${props.projectId}`);
});

const parent = computed(() => {
  if (props.projectId) {
    return `projects/${props.projectId}`;
  }
  return "workspaces/-";
});

const emailSuffix = computed(() => getWorkloadIdentitySuffix(props.projectId));

const isEditMode = computed(
  () =>
    !!props.workloadIdentity &&
    props.workloadIdentity.name !== unknownUser().name
);

const allowUpdate = computed(() => {
  if (!isEditMode.value) {
    return true;
  }
  if (props.projectId && project.value) {
    return hasProjectPermissionV2(
      project.value,
      "bb.workloadIdentities.update"
    );
  }
  return hasWorkspacePermissionV2("bb.workloadIdentities.update");
});

const allowUpdateRoles = computed(() => {
  if (props.projectId && project.value) {
    return hasProjectPermissionV2(project.value, "bb.projects.setIamPolicy");
  }
  return hasWorkspacePermissionV2("bb.workspaces.setIamPolicy");
});

const initialRoles = computed(() => {
  if (
    !props.workloadIdentity ||
    props.workloadIdentity.name === UNKNOWN_USER_NAME
  ) {
    return props.projectId ? [] : [PresetRoleType.WORKSPACE_MEMBER];
  }

  if (props.projectId && project.value) {
    const policy = projectIamPolicyStore.getProjectIamPolicy(
      project.value.name
    );
    const roles = policy.bindings
      .filter((binding: Binding) =>
        binding.members.includes(
          getWorkloadIdentityNameInBinding(props.workloadIdentity!.email)
        )
      )
      .map((binding: Binding) => binding.role);
    return roles;
  }

  const roles = workspaceStore.userMapToRoles.get(
    `workloadIdentities/${props.workloadIdentity.email}`
  );
  return roles ? [...roles] : [];
});

// Parse subject pattern and extract owner/repo/branch/refType
const parseSubjectPattern = (pattern: string) => {
  const { providerType } = state.wif;

  if (providerType === WorkloadIdentityConfig_ProviderType.GITHUB) {
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
  () => props.workloadIdentity,
  (wi) => {
    if (!wi) {
      return;
    }
    state.workloadIdentity = cloneDeep(create(UserSchema, wi));
    state.roles = [...initialRoles.value];

    const config = wi.workloadIdentityConfig;
    if (config) {
      state.wif.emailPrefix = wi.email.split("@")[0];
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

const allowConfirm = computed(() => {
  if (!state.wif.emailPrefix && !state.workloadIdentity.email) {
    return false;
  }
  if (!state.wif.owner) {
    return false;
  }
  if (!state.wif.issuerUrl) {
    return false;
  }
  return true;
});

const updateProjectIamPolicyForMember = async (
  projectName: string,
  member: string,
  roles: string[]
) => {
  const policy = cloneDeep(
    projectIamPolicyStore.getProjectIamPolicy(projectName)
  );

  // Remove member from all existing bindings
  for (const binding of policy.bindings) {
    binding.members = binding.members.filter((m) => m !== member);
  }

  // Remove empty bindings
  policy.bindings = policy.bindings.filter(
    (binding) => binding.members.length > 0
  );

  // Add member to new role bindings
  for (const role of roles) {
    const existingBinding = policy.bindings.find((b) => b.role === role);
    if (existingBinding) {
      if (!existingBinding.members.includes(member)) {
        existingBinding.members.push(member);
      }
    } else {
      policy.bindings.push({
        role,
        members: [member],
        condition: undefined,
        parsedExpr: undefined,
      } as Binding);
    }
  }

  await projectIamPolicyStore.updateProjectIamPolicy(projectName, policy);
};

const createOrUpdateWorkloadIdentity = async () => {
  state.isRequesting = true;
  try {
    if (isEditMode.value) {
      await updateWorkloadIdentity();
    } else {
      await createWorkloadIdentity();
    }
  } catch {
    // nothing
  } finally {
    state.isRequesting = false;
  }
};

const createWorkloadIdentity = async () => {
  const emailPrefix = state.workloadIdentity.email.split("@")[0];
  const wi = await workloadIdentityStore.createWorkloadIdentity(
    emailPrefix,
    {
      title: state.workloadIdentity.title || emailPrefix,
      workloadIdentityConfig: create(WorkloadIdentityConfigSchema, {
        providerType: state.wif.providerType,
        issuerUrl: state.wif.issuerUrl,
        allowedAudiences: state.wif.audience ? [state.wif.audience] : [],
        subjectPattern: state.wif.subjectPattern,
      }),
    },
    parent.value
  );
  const createdUser = workloadIdentityToUser(wi);

  if (state.roles.length > 0) {
    if (props.projectId && project.value) {
      await updateProjectIamPolicyForMember(
        project.value.name,
        getWorkloadIdentityNameInBinding(createdUser.email),
        state.roles
      );
    } else {
      await workspaceStore.patchIamPolicy([
        {
          member: getWorkloadIdentityNameInBinding(createdUser.email),
          roles: state.roles,
        },
      ]);
    }
  }
  emit("created", createdUser);
  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.created"),
  });
  emit("close");
};

const updateWorkloadIdentity = async () => {
  const wi = props.workloadIdentity;
  if (!wi) {
    return;
  }

  const updateMask: string[] = [];
  if (state.workloadIdentity.title !== wi.title) {
    updateMask.push("title");
  }

  let updatedUser: User = wi;

  const updated = await workloadIdentityStore.updateWorkloadIdentity(
    {
      name: ensureWorkloadIdentityFullName(wi.email),
      title: state.workloadIdentity.title,
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
  updatedUser = workloadIdentityToUser(updated);

  if (!isEqual([...initialRoles.value].sort(), [...state.roles].sort())) {
    if (props.projectId && project.value) {
      await updateProjectIamPolicyForMember(
        project.value.name,
        getWorkloadIdentityNameInBinding(updatedUser.email),
        state.roles
      );
    } else {
      await workspaceStore.patchIamPolicy([
        {
          member: getWorkloadIdentityNameInBinding(updatedUser.email),
          roles: state.roles,
        },
      ]);
    }
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
