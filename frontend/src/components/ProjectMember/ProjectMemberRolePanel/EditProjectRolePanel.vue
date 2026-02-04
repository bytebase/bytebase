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
            v-model:value="title"
            type="text"
            :placeholder="displayRoleTitle(binding.role)"
          />
        </div>

        <AddProjectMemberForm
          ref="formRef"
          class="w-full"
          :project-name="projectResourceName"
          :binding="binding"
          :is-edit="true"
          :disable-role-change="true"
          :disable-member-change="true"
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
              {{ $t("common.update") }}
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
import { computed, onMounted, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBButtonConfirm } from "@/bbkit";
import AddProjectMemberForm from "@/components/ProjectMember/AddProjectMember/AddProjectMemberForm.vue";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
} from "@/store";
import { State } from "@/types/proto-es/v1/common_pb";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { displayRoleTitle } from "@/utils";
import { getBindingIdentifier } from "../utils";

const props = defineProps<{
  project: Project;
  binding: Binding;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const title = ref<string>("");
const formRef = ref<InstanceType<typeof AddProjectMemberForm>>();

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
  return props.binding.members.length === 1 && formRef.value?.allowConfirm;
});

onMounted(() => {
  const binding = props.binding;
  // Set the display title with the role name.
  title.value = binding.condition?.title || displayRoleTitle(binding.role);
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
  const binding = await formRef.value?.getBinding(title.value);
  if (!binding) {
    return;
  }
  const member = binding.members[0];

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

  policy.bindings.push(binding);

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
