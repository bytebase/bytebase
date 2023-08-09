<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('common.edit')"
      :closable="true"
      class="w-[30rem] max-w-[100vw] relative"
    >
      <div class="w-full mb-4 pb-4">
        <div class="w-full flex flex-col justify-start items-start gap-y-2">
          <span>{{ $t("common.user") }}</span>
          <UserSelect
            :user="extractUserUID(member.user.name)"
            style="width: 100%"
            :include-all="false"
            :disabled="true"
          />
          <span>{{ $t("common.role.self") }}</span>
          <ProjectMemberRoleSelect
            :role="singleBinding.rawBinding.role"
            :disabled="true"
          />
          <span>{{ $t("common.expiration") }}</span>
          <NDatePicker
            v-model:value="state.expirationTimestamp"
            style="width: 100%"
            type="datetime"
            :is-date-disabled="(date: number) => date < Date.now()"
            clearable
          />
          <span v-if="!state.expirationTimestamp" class="textinfolabel">{{
            $t("project.members.role-never-expires")
          }}</span>
        </div>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" :disabled="!allowConfirm" @click="doConfirm">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { NDrawer, NDrawerContent, NButton, NDatePicker } from "naive-ui";
import { computed, onMounted, reactive } from "vue";
import { useI18n } from "vue-i18n";
import {
  pushNotification,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
} from "@/store";
import { ComposedProject } from "@/types";
import { Expr } from "@/types/proto/google/api/expr/v1alpha1/syntax";
import { Binding } from "@/types/proto/v1/iam_policy";
import { extractUserUID } from "@/utils";
import {
  convertFromExpr,
  stringifyConditionExpression,
} from "@/utils/issue/cel";
import {
  ComposedProjectMember,
  SingleBinding,
} from "../ProjectMemberTable/types";

const props = defineProps<{
  project: ComposedProject;
  member: ComposedProjectMember;
  singleBinding: SingleBinding;
}>();

interface LocalState {
  expirationTimestamp?: number;
}

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const state = reactive<LocalState>({});
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);
const memberName = computed(() => `user:${props.member.user.email}`);
const conditionExpr = computed(() => {
  return convertFromExpr(
    props.singleBinding.rawBinding?.parsedExpr?.expr || Expr.fromPartial({})
  );
});
const allowConfirm = computed(() => {
  if (!conditionExpr.value.expiredTime) {
    return state.expirationTimestamp !== undefined;
  }
  return (
    new Date(conditionExpr.value.expiredTime).getTime() !==
    state.expirationTimestamp
  );
});

onMounted(() => {
  if (props.singleBinding.expiration) {
    state.expirationTimestamp = props.singleBinding.expiration.getTime();
  }
});

const doConfirm = async () => {
  const policy = cloneDeep(iamPolicy.value);
  const rawBinding = policy.bindings.find((binding) =>
    isEqual(binding, props.singleBinding.rawBinding)
  );
  if (!rawBinding) {
    return;
  }
  const binding: Binding = {
    ...cloneDeep(rawBinding),
    members: [memberName.value],
  };
  if (binding.condition) {
    binding.condition = {
      ...binding.condition,
      expression: stringifyConditionExpression({
        ...conditionExpr.value,
        expiredTime: state.expirationTimestamp
          ? new Date(state.expirationTimestamp).toISOString()
          : undefined,
      }),
    };
  } else {
    binding.condition = {
      expression: stringifyConditionExpression({
        expiredTime: state.expirationTimestamp
          ? new Date(state.expirationTimestamp).toISOString()
          : undefined,
      }),
      title: "",
      description: "",
      location: "",
    };
  }

  if (rawBinding.members.length > 1) {
    rawBinding.members = rawBinding.members.filter(
      (member) => member !== memberName.value
    );
    policy.bindings.push(binding);
  } else {
    rawBinding.condition = binding.condition;
  }
  await useProjectIamPolicyStore().updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("project.settings.success-member-added-prompt"),
  });
  emit("close");
};
</script>
