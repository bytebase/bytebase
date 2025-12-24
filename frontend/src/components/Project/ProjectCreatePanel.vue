<template>
  <DrawerContent
    :title="$t('quick-action.create-project')"
    class="w-[40rem] max-w-[100vw]"
  >
    <div class="w-full">
      <div class="grid gap-y-6 gap-x-4 grid-cols-1">
        <div class="col-span-1">
          <label
            for="name"
            class="text-base leading-6 font-medium text-control"
          >
            {{ $t("project.create-modal.project-name") }}
            <RequiredStar />
          </label>
          <BBTextField
            v-model:value="state.project.title"
            class="mt-2 mb-1 w-full"
            :required="true"
            :placeholder="$t('project.create-modal.project-name')"
            :maxlength="200"
          />
          <ResourceIdField
            ref="resourceIdField"
            editing-class="mt-2"
            resource-type="project"
            :value="state.resourceId"
            :resource-title="state.project.title"
            :fetch-resource="
              (id) =>
                projectV1Store.getOrFetchProjectByName(
                  `${projectNamePrefix}${id}`,
                  true /* silent */
                )
            "
            @update:value="state.resourceId = $event"
          />
        </div>
      </div>
    </div>

    <div
      v-if="state.isCreating"
      class="absolute inset-0 bg-white/50 flex justify-center items-center"
    >
      <BBSpin />
    </div>

    <template #footer>
      <div class="flex justify-end items-center gap-x-3">
        <NButton quaternary @click.prevent="cancel">
          {{ $t("common.cancel") }}
        </NButton>
        <NButton
          type="primary"
          :disabled="!allowCreate"
          @click.prevent="create"
        >
          {{ $t("common.create") }}
        </NButton>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { isEmpty } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { useRouter } from "vue-router";
import { BBSpin, BBTextField } from "@/bbkit";
import RequiredStar from "@/components/RequiredStar.vue";
import { DrawerContent } from "@/components/v2";
import ResourceIdField from "@/components/v2/Form/ResourceIdField.vue";
import { pushNotification, useUIStateStore } from "@/store";
import { projectNamePrefix } from "@/store/modules/v1/common";
import { useProjectV1Store } from "@/store/modules/v1/project";
import { emptyProject } from "@/types";
import type { Project } from "@/types/proto-es/v1/project_service_pb";
import { hasWorkspacePermissionV2 } from "@/utils";

interface LocalState {
  project: Project;
  resourceId: string;
  isCreating: boolean;
}

const props = defineProps<{
  onCreated?: (project: Project) => void;
}>();

const emit = defineEmits<{
  (event: "dismiss"): void;
}>();

const router = useRouter();
const { t } = useI18n();
const projectV1Store = useProjectV1Store();

const state = reactive<LocalState>({
  project: {
    ...emptyProject(),
    title: "New Project",
  },
  resourceId: "",
  isCreating: false,
});
const resourceIdField = ref<InstanceType<typeof ResourceIdField>>();

const allowCreate = computed(() => {
  if (isEmpty(state.project.title)) return false;
  if (!resourceIdField.value?.isValidated) return false;
  if (!hasWorkspacePermissionV2("bb.projects.create")) {
    return false;
  }
  return true;
});

const create = async () => {
  if (!allowCreate.value) {
    return;
  }

  try {
    state.isCreating = true;
    const createdProject = await projectV1Store.createProject(
      state.project,
      state.resourceId
    );

    if (props.onCreated) {
      props.onCreated(createdProject);
    } else {
      useUIStateStore().saveIntroStateByKey({
        key: "project.visit",
        newState: true,
      });

      const url = {
        path: `/${createdProject.name}`,
      };
      router.push(url);
      emit("dismiss");
    }

    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.create-modal.success-prompt", {
        name: createdProject.title,
      }),
    });
  } finally {
    state.isCreating = false;
  }
};

const cancel = () => {
  emit("dismiss");
};
</script>
