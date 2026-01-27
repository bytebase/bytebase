<template>
  <Drawer @close="$emit('close')">
    <DrawerContent
      class="w-[40rem] max-w-[100vw]"
      :title="
        isEditMode
          ? $t('settings.members.update-service-account')
          : $t('settings.members.add-service-account')
      "
    >
      <template #default>
        <div class="flex flex-col gap-y-6">
          <div class="flex flex-col gap-y-2">
            <label class="block text-sm font-medium leading-5 text-control">
              {{ $t("common.name") }}
            </label>
            <NInput
              v-model:value="state.serviceAccount.title"
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
              v-model:value="state.serviceAccount.email"
              :domain="emailSuffix"
              :show-domain="true"
              :disabled="isEditMode"
            />
          </div>

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
            @click="createOrUpdateServiceAccount"
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
import { NButton, NInput } from "naive-ui";
import { computed, reactive, watch } from "vue";
import { useI18n } from "vue-i18n";
import EmailInput from "@/components/EmailInput.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import { RoleSelect } from "@/components/v2/Select";
import {
  ensureServiceAccountFullName,
  pushNotification,
  serviceAccountToUser,
  useProjectIamPolicyStore,
  useProjectV1Store,
  useServiceAccountStore,
  useWorkspaceV1Store,
} from "@/store";
import {
  getServiceAccountNameInBinding,
  getServiceAccountSuffix,
  UNKNOWN_USER_NAME,
  unknownUser,
} from "@/types";
import { PresetRoleType } from "@/types/iam";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { User } from "@/types/proto-es/v1/user_service_pb";
import { UserSchema } from "@/types/proto-es/v1/user_service_pb";
import { hasProjectPermissionV2, hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  isRequesting: boolean;
  serviceAccount: User;
  roles: string[];
}

const props = defineProps<{
  serviceAccount?: User;
  projectId?: string;
}>();

const emit = defineEmits<{
  (event: "close"): void;
  (event: "created", user: User): void;
  (event: "updated", user: User): void;
}>();

const { t } = useI18n();
const serviceAccountStore = useServiceAccountStore();
const workspaceStore = useWorkspaceV1Store();
const projectStore = useProjectV1Store();
const projectIamPolicyStore = useProjectIamPolicyStore();

const state = reactive<LocalState>({
  isRequesting: false,
  serviceAccount: unknownUser(),
  roles: props.projectId ? [] : [PresetRoleType.WORKSPACE_MEMBER],
});

const project = computed(() => {
  if (!props.projectId) return undefined;
  return projectStore.getProjectByName(`projects/${props.projectId}`);
});

const parent = computed(() => {
  if (props.projectId) {
    return `projects/${props.projectId}`;
  }
  return undefined;
});

const emailSuffix = computed(() => getServiceAccountSuffix(props.projectId));

const isEditMode = computed(
  () =>
    !!props.serviceAccount && props.serviceAccount.name !== unknownUser().name
);

const allowUpdate = computed(() => {
  if (!isEditMode.value) {
    return true;
  }
  if (props.projectId && project.value) {
    return hasProjectPermissionV2(project.value, "bb.serviceAccounts.update");
  }
  return hasWorkspacePermissionV2("bb.serviceAccounts.update");
});

const allowUpdateRoles = computed(() => {
  if (props.projectId && project.value) {
    return hasProjectPermissionV2(project.value, "bb.projects.setIamPolicy");
  }
  return hasWorkspacePermissionV2("bb.workspaces.setIamPolicy");
});

const initialRoles = computed(() => {
  if (
    !props.serviceAccount ||
    props.serviceAccount.name === UNKNOWN_USER_NAME
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
          getServiceAccountNameInBinding(props.serviceAccount!.email)
        )
      )
      .map((binding: Binding) => binding.role);
    return roles;
  }

  const roles = workspaceStore.userMapToRoles.get(
    `serviceAccounts/${props.serviceAccount.email}`
  );
  return roles ? [...roles] : [];
});

watch(
  () => props.serviceAccount,
  (sa) => {
    if (!sa) {
      return;
    }
    state.serviceAccount = cloneDeep(create(UserSchema, sa));
    state.roles = [...initialRoles.value];
  },
  {
    immediate: true,
  }
);

const allowConfirm = computed(() => {
  if (!state.serviceAccount.email) {
    return false;
  }
  return true;
});

const extractTitle = (email: string): string => {
  const atIndex = email.indexOf("@");
  if (atIndex !== -1) {
    return email.substring(0, atIndex);
  }
  return email;
};

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

const createOrUpdateServiceAccount = async () => {
  state.isRequesting = true;
  try {
    if (isEditMode.value) {
      await updateServiceAccount();
    } else {
      await createServiceAccount();
    }
  } catch {
    // nothing
  } finally {
    state.isRequesting = false;
  }
};

const createServiceAccount = async () => {
  const serviceAccountId = state.serviceAccount.email.split("@")[0];
  const sa = await serviceAccountStore.createServiceAccount(
    serviceAccountId,
    {
      title:
        state.serviceAccount.title || extractTitle(state.serviceAccount.email),
    },
    parent.value
  );
  const createdUser = serviceAccountToUser(sa);

  if (state.roles.length > 0) {
    if (props.projectId && project.value) {
      await updateProjectIamPolicyForMember(
        project.value.name,
        getServiceAccountNameInBinding(createdUser.email),
        state.roles
      );
    } else {
      await workspaceStore.patchIamPolicy([
        {
          member: getServiceAccountNameInBinding(createdUser.email),
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

const updateServiceAccount = async () => {
  const sa = props.serviceAccount;
  if (!sa) {
    return;
  }

  const updateMask: string[] = [];
  if (state.serviceAccount.title !== sa.title) {
    updateMask.push("title");
  }

  let updatedUser: User = sa;

  if (updateMask.length > 0) {
    const updated = await serviceAccountStore.updateServiceAccount(
      {
        name: ensureServiceAccountFullName(sa.email),
        title: state.serviceAccount.title,
      },
      create(FieldMaskSchema, {
        paths: [...updateMask],
      })
    );
    updatedUser = serviceAccountToUser(updated);
  }

  if (!isEqual([...initialRoles.value].sort(), [...state.roles].sort())) {
    if (props.projectId && project.value) {
      await updateProjectIamPolicyForMember(
        project.value.name,
        getServiceAccountNameInBinding(updatedUser.email),
        state.roles
      );
    } else {
      await workspaceStore.patchIamPolicy([
        {
          member: getServiceAccountNameInBinding(updatedUser.email),
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
