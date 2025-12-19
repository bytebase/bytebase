<template>
  <Drawer
    :show="true"
    width="auto"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <DrawerContent
      :title="$t('settings.members.grant-access')"
      :closable="true"
      class="w-200 max-w-[100vw] relative"
    >
      <div
        v-for="(binding, index) in state.bindings"
        :key="index"
        class="w-full"
      >
        <AddProjectMemberForm
          v-if="binding"
          ref="formRefs"
          class="w-full border-b mb-4 pb-4"
          :project-name="project.name"
          :binding="binding"
          :allow-remove="state.bindings.length > 1"
          @remove="handleRemove(index)"
        />
      </div>
      <div>
        <NButton @click="handleAddMore">
          <heroicons-solid:plus class="w-5 h-auto text-gray-400" />
          {{ $t("project.members.add-more") }}
        </NButton>
      </div>
      <template #footer>
        <div class="flex items-center justify-end gap-x-2">
          <NButton @click="$emit('close')">{{ $t("common.cancel") }}</NButton>
          <NButton type="primary" :disabled="!allowConfirm" @click="addMembers">
            {{ $t("common.confirm") }}
          </NButton>
        </div>
      </template>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { Drawer, DrawerContent } from "@/components/v2";
import {
  pushNotification,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
} from "@/store";
import { PresetRoleType } from "@/types";
import type { Binding } from "@/types/proto-es/v1/iam_policy_pb";
import { BindingSchema } from "@/types/proto-es/v1/iam_policy_pb";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { getBindingIdentifier } from "../utils";
import AddProjectMemberForm from "./AddProjectMemberForm.vue";

const props = defineProps<{
  project: Project;
  bindings?: Binding[];
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

interface LocalState {
  bindings: Binding[];
}

const { t } = useI18n();
const state = reactive<LocalState>({
  bindings: props.bindings || [
    create(BindingSchema, {
      role: PresetRoleType.PROJECT_VIEWER,
    }),
  ],
});
const formRefs = ref<InstanceType<typeof AddProjectMemberForm>[]>([]);
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const allowConfirm = computed(() => {
  // Check if all forms are completed.
  for (const form of formRefs.value) {
    if (!form) {
      continue;
    }
    if (!form.allowConfirm) {
      return false;
    }
  }
  return true;
});

const handleAddMore = () => {
  state.bindings.push(
    create(BindingSchema, {
      role: PresetRoleType.PROJECT_VIEWER,
    })
  );
};

const handleRemove = (index: number) => {
  state.bindings.splice(index, 1);
};

const mergePolicyBinding = async () => {
  const bindingMap = new Map<string, Binding>();
  for (const formRef of formRefs.value) {
    const binding = await formRef.getBinding();
    const key = getBindingIdentifier(binding);
    if (!bindingMap.has(key)) {
      bindingMap.set(key, binding);
    } else {
      bindingMap.get(key)?.members?.push(...binding.members);
    }
  }

  const policy = cloneDeep(iamPolicy.value);
  for (const binding of policy.bindings) {
    const key = getBindingIdentifier(binding);
    if (bindingMap.has(key)) {
      binding.members = [
        ...new Set([
          ...binding.members,
          ...(bindingMap.get(key)?.members ?? []),
        ]),
      ];
      bindingMap.delete(key);
    }
  }

  for (const binding of bindingMap.values()) {
    policy.bindings.push({
      ...binding,
      members: [...new Set(binding.members)],
    });
  }

  return policy;
};

const addMembers = async () => {
  if (!allowConfirm.value) {
    return;
  }

  const policy = await mergePolicyBinding();
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
