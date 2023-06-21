<template>
  <NDrawer
    :show="true"
    width="auto"
    :auto-focus="false"
    @update:show="(show: boolean) => !show && $emit('close')"
  >
    <NDrawerContent
      :title="$t('project.members.add-member')"
      :closable="true"
      class="w-[30rem] max-w-[100vw] relative"
    >
      <div
        v-for="(binding, index) in state.bindings"
        :key="index"
        class="w-full"
      >
        <AddProjectMemberForm
          v-if="binding"
          class="w-full border-b mb-4 pb-4"
          :project="project"
          :binding="binding"
          :allow-remove="filteredBindings.length > 1"
          @remove="handleRemove(binding, index)"
        />
      </div>
      <div>
        <NButton @click="handleAddMore">
          <heroicons-solid:plus class="w-5 h-auto" />{{
            $t("project.members.add-more")
          }}
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
    </NDrawerContent>
  </NDrawer>
</template>

<script lang="ts" setup>
import { NDrawer, NDrawerContent, NButton } from "naive-ui";
import { ComposedProject } from "@/types";
import { Binding } from "@/types/proto/v1/project_service";
import { computed, onMounted } from "vue";
import { reactive } from "vue";
import AddProjectMemberForm from "./AddProjectMemberForm.vue";
import { cloneDeep } from "lodash-es";
import {
  pushNotification,
  useProjectIamPolicy,
  useProjectIamPolicyStore,
} from "@/store";
import { useI18n } from "vue-i18n";

const props = defineProps<{
  project: ComposedProject;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

interface LocalState {
  bindings: (Binding | undefined)[];
}

const { t } = useI18n();
const state = reactive<LocalState>({
  bindings: [],
});
const projectResourceName = computed(() => props.project.name);
const { policy: iamPolicy } = useProjectIamPolicy(projectResourceName);

const filteredBindings = computed(() => {
  return state.bindings.filter((binding) => binding !== undefined) as Binding[];
});

const allowConfirm = computed(() => {
  for (const binding of filteredBindings.value) {
    if (binding.members.length === 0 || binding.role === "") return false;
  }
  return true;
});

onMounted(() => {
  state.bindings = [Binding.fromPartial({})];
});

const handleAddMore = () => {
  state.bindings.push(Binding.fromPartial({}));
};

const handleRemove = (binding: Binding, index: number) => {
  state.bindings[index] = undefined;
};

const addMembers = async () => {
  if (!allowConfirm.value) {
    return;
  }

  const policy = cloneDeep(iamPolicy.value);
  policy.bindings.push(...filteredBindings.value);
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
