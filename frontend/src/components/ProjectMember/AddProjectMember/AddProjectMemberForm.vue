<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-4">
    <MembersBindingSelect
      v-model:value="state.memberList"
      :required="true"
      :include-all-users="true"
      :disabled="disableMemberChange"
    >
      <template #suffix>
        <NButton v-if="allowRemove" text @click="$emit('remove')">
          <template #icon>
            <heroicons:trash class="w-4 h-4" />
          </template>
        </NButton>
      </template>
    </MembersBindingSelect>

    <div class="w-full flex flex-col gap-y-2">
      <div class="flex items-center gap-x-1">
        <span>{{ $t("settings.members.assign-role") }}</span>
        <RequiredStar />
      </div>
      <BBAttention
        v-if="requiredPermissions"
        :type="'info'"
        class="w-full"
      >
        <div>
          {{ $t("project.members.roles-are-filtered-by-permissions") }}
          <ul class="list-disc pl-4">
            <li v-for="permission in requiredPermissions" :key="permission">
              {{ permission }}
            </li>
          </ul>
        </div>
      </BBAttention>
      <RoleSelect
        v-model:value="state.role"
        :include-workspace-roles="false"
        :suffix="''"
        :filter="filterRole"
      />
    </div>
    <div v-if="selectedRole" class="w-full flex flex-col gap-y-2">
      <div class="flex items-center gap-x-1">
        <span>
          {{ $t("common.permissions") }}
          ({{
              selectedRole.permissions.length
          }})
        </span>
      </div>
      <div class="max-h-[10em] overflow-auto border rounded-sm p-2">
        <p
          v-for="permission in selectedRole.permissions"
          :key="permission"
          class="text-sm leading-5"
        >
          {{ permission }}
        </p>
      </div>
    </div>
    <div class="w-full flex flex-col gap-y-2">
      <div class="flex items-center gap-x-1">
        <span>{{ $t("common.reason") }}</span>
        <RequiredStar v-if="requireReason" />
      </div>
      <NInput
        v-model:value="state.reason"
        type="textarea"
        rows="2"
        :placeholder="`${$t('common.reason')} ${requireReason ? '' : `(${$t('common.optional')})`}`"
      />
    </div>
    <div
      v-if="
        roleHasDatabaseLimitation(state.role)
      "
      class="w-full flex flex-col gap-y-2"
    >
      <div class="flex items-center gap-x-1">
        <span>{{ $t("common.databases") }}</span>
        <RequiredStar />
      </div>
      <QuerierDatabaseResourceForm
        ref="databaseResourceFormRef"
        :database-resources="databaseResources"
        :project-name="projectName"
        :required-feature="PlanFeature.FEATURE_IAM"
        :include-cloumn="false"
      />
    </div>

    <div class="w-full flex flex-col gap-y-2">
      <div class="flex items-center gap-x-1">
        <span>{{ $t("common.expiration") }}</span>
        <RequiredStar />
      </div>
      <ExpirationSelector
        ref="expirationSelectorRef"
        :role="state.role"
        v-model:timestamp-in-ms="state.expirationTimestampInMS"
        class="grid-cols-3 sm:grid-cols-4"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { NButton, NInput } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { BBAttention } from "@/bbkit";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import QuerierDatabaseResourceForm from "@/components/GrantRequestPanel/DatabaseResourceForm/index.vue";
import MembersBindingSelect from "@/components/Member/MembersBindingSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { RoleSelect } from "@/components/v2/Select";
import { useRoleStore } from "@/store";
import { type DatabaseResource } from "@/types";
import { type Binding, BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Role } from "@/types/proto-es/v1/role_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { buildConditionExpr } from "@/utils/issue/cel";
import { roleHasDatabaseLimitation } from "../utils";

const props = withDefaults(
  defineProps<{
    projectName: string;
    binding: Binding;
    allowRemove: boolean;
    requireReason?: boolean;
    disableMemberChange?: boolean;
    requiredPermissions?: string[];
    databaseResources?: DatabaseResource[];
  }>(),
  {
    disableMemberChange: false,
    requireReason: false,
    databaseResources: () => [],
  }
);

defineEmits<{
  (event: "remove"): void;
}>();

interface LocalState {
  memberList: string[];
  role: string;
  reason: string;
  expirationTimestampInMS?: number;
  databaseId?: string;
}

const filterRole = (role: Role) => {
  if (props.requiredPermissions) {
    return props.requiredPermissions.every((p) => role.permissions.includes(p));
  }
  return true;
};

const getInitialState = (): LocalState => {
  const defaultState: LocalState = {
    role: props.binding.role,
    memberList: props.binding.members,
    reason: "",
  };

  return defaultState;
};

const state = reactive<LocalState>(getInitialState());
const expirationSelectorRef = ref<InstanceType<typeof ExpirationSelector>>();
const databaseResourceFormRef =
  ref<InstanceType<typeof QuerierDatabaseResourceForm>>();
const roleStore = useRoleStore();

const selectedRole = computed(() => roleStore.getRoleByName(state.role));

defineExpose({
  reason: computed(() => state.reason),
  getDatabaseResources: async () => {
    const resources =
      await databaseResourceFormRef.value?.getDatabaseResources();
    return resources;
  },
  getBinding: async (): Promise<Binding> => {
    const databaseResources =
      await databaseResourceFormRef.value?.getDatabaseResources();
    const condition = buildConditionExpr({
      role: state.role,
      description: state.reason,
      expirationTimestampInMS: state.expirationTimestampInMS,
      databaseResources,
    });

    return create(BindingSchema, {
      members: state.memberList,
      role: state.role,
      condition,
    });
  },
  expirationTimestampInMS: computed(() => state.expirationTimestampInMS),
  allowConfirm: computed(() => {
    if (!state.role) {
      return false;
    }
    if (state.memberList.length <= 0) {
      return false;
    }
    if (!expirationSelectorRef.value?.isValid) {
      return false;
    }
    // Only check database form validity if it's rendered
    if (
      databaseResourceFormRef.value &&
      !databaseResourceFormRef.value.isValid
    ) {
      return false;
    }
    if (props.requireReason && !state.reason.trim()) {
      return false;
    }
    return true;
  }),
});
</script>
