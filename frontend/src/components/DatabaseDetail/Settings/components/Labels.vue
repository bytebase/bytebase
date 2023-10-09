<template>
  <div class="space-y-4">
    <div class="flex items-center">
      <div class="flex-1">
        <div class="flex items-center">
          <p class="text-lg font-medium leading-7 text-main flex">
            {{ $t("common.labels") }}
          </p>
        </div>
      </div>
    </div>
    <div>
      <LabelListEditor
        ref="labelListEditorRef"
        v-model:kv-list="state.kvList"
        :readonly="!allowAdmin"
        :show-errors="dirty"
        class="max-w-[30rem]"
      />
    </div>
    <div
      v-if="allowAdmin"
      class="flex flex-row justify-end items-center gap-x-3"
    >
      <NButton v-if="dirty" @click="handleCancel">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton
        v-if="dirty"
        :disabled="!allowSave || state.isUpdating"
        :loading="state.isUpdating"
        type="primary"
        @click="handleSave"
      >
        {{ $t("common.save") }}
      </NButton>
    </div>
  </div>
</template>

<script setup lang="ts">
import { cloneDeep, isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { LabelListEditor } from "@/components/Label/";
import {
  pushNotification,
  useCurrentUserV1,
  useDatabaseV1Store,
} from "@/store";
import { type ComposedDatabase } from "@/types";
import { Database } from "@/types/proto/v1/database_service";
import {
  PRESET_LABEL_KEYS,
  convertKVListToLabels,
  convertLabelsToKVList,
  hasPermissionInProjectV1,
  hasWorkspacePermissionV1,
} from "@/utils";

type LocalState = {
  kvList: { key: string; value: string }[];
  isUpdating: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
}>();

const { t } = useI18n();
const labelListEditorRef = ref<InstanceType<typeof LabelListEditor>>();
const me = useCurrentUserV1();
const state = reactive<LocalState>({
  kvList: [],
  isUpdating: false,
});

const allowAdmin = computed(() => {
  const project = props.database.projectEntity;
  return (
    hasWorkspacePermissionV1(
      "bb.permission.workspace.manage-label",
      me.value.userRole
    ) ||
    hasPermissionInProjectV1(
      project.iamPolicy,
      me.value,
      "bb.permission.project.manage-general"
    )
  );
});

const convert = () => {
  const labels = cloneDeep(props.database.labels);
  // Pre-fill preset label keys with empty values
  for (const key of PRESET_LABEL_KEYS) {
    if (!(key in labels)) {
      labels[key] = "";
    }
  }
  return convertLabelsToKVList(labels, true /* sort */);
};

const dirty = computed(() => {
  const original = convert();
  const local = state.kvList;
  return !isEqual(original, local);
});

watch(
  () => props.database.labels,
  () => {
    state.kvList = convert();
  },
  {
    immediate: true,
    deep: true,
  }
);

const allowSave = computed(() => {
  const errors = labelListEditorRef.value?.flattenErrors ?? [];
  return errors.length === 0;
});

const handleCancel = () => {
  state.kvList = convert();
};

const handleSave = async () => {
  if (!allowSave.value) return;
  state.isUpdating = true;
  try {
    // Won't omit empty `bb.tenant` value.
    // Otherwise the server API won't update the labels field correctly.
    const labels = convertKVListToLabels(state.kvList, false /* !omitEmpty */);

    const patch = {
      ...Database.fromPartial(props.database),
      labels,
    };
    await useDatabaseV1Store().updateDatabase({
      database: patch,
      updateMask: ["labels"],
    });
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.isUpdating = false;
  }
};
</script>
