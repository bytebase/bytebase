<template>
  <div class="w-full flex flex-col justify-start items-start gap-y-4">
    <MembersBindingSelect
      v-model:value="state.memberList"
      :required="true"
      :include-all-users="true"
      :include-service-account="true"
      :include-workload-identity="true"
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
      <RoleSelect
        v-model:value="state.role"
        :include-workspace-roles="false"
        :suffix="''"
        :filter="filterRole"
      />
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
        state.role !== PresetRoleType.PROJECT_OWNER &&
        checkRoleContainsAnyPermission(state.role, 'bb.sql.select')
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
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import QuerierDatabaseResourceForm from "@/components/GrantRequestPanel/DatabaseResourceForm/index.vue";
import MembersBindingSelect from "@/components/Member/MembersBindingSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { RoleSelect } from "@/components/v2/Select";
import { type DatabaseResource, PresetRoleType } from "@/types";
import { type Binding, BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { checkRoleContainsAnyPermission } from "@/utils";
import { buildConditionExpr } from "@/utils/issue/cel";

const props = withDefaults(
  defineProps<{
    projectName: string;
    binding: Binding;
    allowRemove: boolean;
    requireReason?: boolean;
    disableMemberChange?: boolean;
    supportRoles?: string[];
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

const filterRole = (role: string) => {
  if (!props.supportRoles) {
    return true;
  }
  return props.supportRoles.includes(role);
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
