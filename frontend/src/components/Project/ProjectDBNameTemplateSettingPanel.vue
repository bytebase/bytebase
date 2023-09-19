<template>
  <div v-if="show">
    <div class="text-lg font-medium leading-7 text-main">
      {{ $t("project.db-name-template") }}
    </div>
    <div class="textinfolabel">
      <i18n-t keypath="label.db-name-template-tips">
        <template #placeholder>
          <!-- prettier-ignore -->
          <code v-pre class="text-xs font-mono bg-control-bg">{{DB_NAME}}__{{TENANT}}</code>
        </template>
        <template #link>
          <a
            class="normal-link inline-flex items-center"
            href="https://www.bytebase.com/docs/change-database/batch-change/#specify-database-name-template"
            target="__BLANK"
          >
            {{ $t("common.learn-more") }}
            <heroicons-outline:external-link class="w-4 h-4 ml-1" />
          </a>
        </template>
      </i18n-t>
    </div>
    <div class="mt-3 space-y-2">
      <div>
        <input
          v-model="state.dbNameTemplate"
          type="text"
          class="textfield w-full"
          :disabled="!state.isEditingDBNameTemplate"
        />
      </div>
      <div class="flex items-center justify-end gap-x-2">
        <button
          v-if="!state.isEditingDBNameTemplate"
          :disabled="!allowEdit"
          class="btn-normal"
          @click="beginEditDBNameTemplate"
        >
          {{ $t("common.edit") }}
        </button>
        <template v-if="state.isEditingDBNameTemplate">
          <button class="btn-normal" @click="cancelEditDBNameTemplate">
            {{ $t("common.cancel") }}
          </button>
          <button
            class="btn-primary"
            :disabled="!allowUpdateDBNameTemplate"
            @click="confirmEditDBNameTemplate"
          >
            {{ $t("common.update") }}
          </button>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep } from "lodash-es";
import { PropType, computed, reactive } from "vue";
import { useI18n } from "vue-i18n";
import { pushNotification, useProjectV1Store } from "@/store";
import { Project, TenantMode } from "@/types/proto/v1/project_service";

type LocalState = {
  isEditingDBNameTemplate: boolean;
  dbNameTemplate: string;
};

const props = defineProps({
  project: {
    required: true,
    type: Object as PropType<Project>,
  },
  allowEdit: {
    default: true,
    type: Boolean,
  },
});

const { t } = useI18n();

const state = reactive<LocalState>({
  isEditingDBNameTemplate: false,
  dbNameTemplate: props.project.dbNameTemplate,
});

const show = computed(() => {
  return props.project.tenantMode === TenantMode.TENANT_MODE_ENABLED;
});

const allowUpdateDBNameTemplate = computed(() => {
  return state.dbNameTemplate !== props.project.dbNameTemplate;
});

const beginEditDBNameTemplate = () => {
  state.dbNameTemplate = props.project.dbNameTemplate;
  state.isEditingDBNameTemplate = true;
};

const cancelEditDBNameTemplate = () => {
  state.dbNameTemplate = props.project.dbNameTemplate;
  state.isEditingDBNameTemplate = false;
};

const confirmEditDBNameTemplate = async () => {
  try {
    const projectPatch = cloneDeep(props.project);
    projectPatch.dbNameTemplate = state.dbNameTemplate;
    const updateMask = ["db_name_template"];
    await useProjectV1Store().updateProject(projectPatch, updateMask);
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("project.successfully-updated-db-name-template"),
    });
  } catch {
    state.dbNameTemplate = props.project.dbNameTemplate;
  } finally {
    state.isEditingDBNameTemplate = false;
  }
};
</script>
