<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="panelTitle"
      :closable="true"
      class="w-5xl max-w-[100vw] relative"
    >
      <div class="w-full flex flex-col justify-start items-start gap-y-4 pb-12">
        <div class="w-full">
          <p class="mb-2">
            <span>{{ $t("project.members.condition-name") }}</span>
          </p>
          <NInput
            v-model:value="state.title"
            type="text"
            :placeholder="displayRoleTitle(binding.role)"
          />
        </div>

        <template v-if="!state.isLoading">
          <div
            v-if="
              binding.role !== PresetRoleType.PROJECT_OWNER &&
              checkRoleContainsAnyPermission(binding.role, 'bb.sql.select')
            "
            class="w-full"
          >
            <div class="flex items-center gap-x-1 mb-2">
              <span>{{ $t("common.databases") }}</span>
              <RequiredStar />
            </div>
            <QuerierDatabaseResourceForm
              ref="databaseResourceFormRef"
              :database-resources="databaseResources"
              :project-name="project.name"
              :include-cloumn="false"
              :required-feature="PlanFeature.FEATURE_IAM"
            />
          </div>
        </template>

        <div class="w-full">
          <p class="mb-2">{{ $t("common.description") }}</p>
          <NInput
            v-model:value="state.description"
            type="textarea"
            :placeholder="$t('project.members.role-description')"
          />
        </div>

        <div class="w-full">
          <p class="mb-2">{{ $t("common.expiration") }}</p>
          <ExpirationSelector
            v-if="!state.isLoading"
            ref="expirationSelectorRef"
            :role="binding.role"
            v-model:timestamp-in-ms="state.expirationTimestamp"
            class="grid-cols-3 sm:grid-cols-4"
          />
        </div>

        <MembersBindingSelect
          :value="binding.members"
          :disabled="true"
          :required="true"
          :allow-change-type="false"
          :include-all-users="true"
          :include-service-account="true"
        />
      </div>
      <template #footer>
        <div class="w-full flex flex-row justify-between items-center">
          <div>
            <BBButtonConfirm
              v-if="allowRemoveRole()"
              :type="'DELETE'"
              :button-text="$t('common.delete')"
              :require-confirm="true"
              @confirm="handleDeleteRole"
            />
          </div>
          <div class="flex items-center justify-end gap-x-2">
            <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
            <NButton
              type="primary"
              :disabled="!allowConfirm"
              @click="handleUpdateRole"
            >
              {{ $t("common.ok") }}
            </NButton>
          </div>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { cloneDeep, isEqual } from "lodash-es";
import { NButton, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import ExpirationSelector from "@/components/ExpirationSelector.vue";
import QuerierDatabaseResourceForm from "@/components/GrantRequestPanel/DatabaseResourceForm/index.vue";
import MembersBindingSelect from "@/components/Member/MembersBindingSelect.vue";
import RequiredStar from "@/components/RequiredStar.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
} from "@/store";
import { PresetRoleType } from "@/types";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { PlanFeature } from "@/types/proto-es/v1/subscription_service_pb";
import { checkRoleContainsAnyPermission, displayRoleTitle } from "@/utils";
import { buildConditionExpr, convertFromExpr } from "@/utils/issue/cel";
import { getBindingIdentifier } from "../utils";

const props = defineProps<{
  project: Project;
  binding: Binding;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

interface LocalState {
  title: string;
  description: string;
  expirationTimestamp?: number;
  isLoading: boolean;
}

const { t } = useI18n();

const state = reactive<LocalState>({
  title: "",
  description: "",
  isLoading: true,
});
const expirationSelectorRef = ref<InstanceType<typeof ExpirationSelector>>();
const databaseResourceFormRef =
  ref<InstanceType<typeof QuerierDatabaseResourceForm>>();

const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const panelTitle = computed(() => {
  return displayRoleTitle(props.binding.role);
});

const allowRemoveRole = () => {
  if (props.project.state === State.DELETED) {
    return false;
  }

  // Don't allow to remove the role if the condition is empty.
  // * No expiration time.
  if (props.binding.condition?.expression === "") {
    return false;
  }

  return true;
};

const allowConfirm = computed(() => {
  // only allow update current single user.
  return (
    props.binding.members.length === 1 &&
    expirationSelectorRef.value?.isValid &&
    databaseResourceFormRef.value?.isValid
  );
});

const databaseResources = computed(() => {
  if (props.binding.parsedExpr) {
    const conditionExpr = convertFromExpr(props.binding.parsedExpr);
    return conditionExpr.databaseResources;
  }
  return undefined;
});

onMounted(() => {
  const binding = props.binding;
  // Set the display title with the role name.
  state.title = binding.condition?.title || "";
  state.description = binding.condition?.description || "";

  if (binding.parsedExpr) {
    const conditionExpr = convertFromExpr(binding.parsedExpr);
    if (conditionExpr.expiredTime) {
      state.expirationTimestamp = new Date(conditionExpr.expiredTime).getTime();
    }
  }

  state.isLoading = false;
});

const handleDeleteRole = async () => {
  const policy = cloneDeep(iamPolicy.value);
  policy.bindings = policy.bindings.filter(
    (binding) => !isEqual(binding, props.binding)
  );
  await useProjectIamPolicyStore().updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );
  emit("close");
};

const handleUpdateRole = async () => {
  const member = props.binding.members[0];

  const newBinding = cloneDeep(props.binding);
  newBinding.members = [member];

  const databaseResources =
    await databaseResourceFormRef.value?.getDatabaseResources();
  newBinding.condition = buildConditionExpr({
    title: state.title,
    role: props.binding.role,
    description: state.description,
    expirationTimestampInMS: state.expirationTimestamp,
    databaseResources,
  });

  const policy = cloneDeep(iamPolicy.value);
  const oldBindingIndex = policy.bindings.findIndex(
    (binding) =>
      getBindingIdentifier(binding) === getBindingIdentifier(props.binding)
  );

  if (oldBindingIndex >= 0) {
    policy.bindings[oldBindingIndex].members = policy.bindings[
      oldBindingIndex
    ].members.filter((m) => m !== member);
    if (policy.bindings[oldBindingIndex].members.length === 0) {
      policy.bindings.splice(oldBindingIndex, 1);
    }
  }

  policy.bindings.push(newBinding);

  await useProjectIamPolicyStore().updateProjectIamPolicy(
    projectResourceName.value,
    policy
  );

  pushNotification({
    module: "bytebase",
    style: "SUCCESS",
    title: t("common.updated"),
  });

  emit("close");
};
</script>
