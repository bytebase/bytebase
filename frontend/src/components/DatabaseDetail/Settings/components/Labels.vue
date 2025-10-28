<template>
  <div class="space-y-4">
    <div class="flex items-center">
      <div class="flex-1">
        <div class="flex items-center">
          <p class="text-lg font-medium leading-7 text-main flex">
            {{ $t("database.labels") }}
          </p>
        </div>
      </div>
    </div>
    <div>
      <LabelListEditor
        ref="labelListEditorRef"
        v-model:kv-list="state.kvList"
        :readonly="!allowEdit"
        :show-errors="dirty"
        class="max-w-[30rem]"
      />
    </div>
    <div v-if="dirty" class="flex flex-row justify-end items-center gap-x-3">
      <NButton @click="handleCancel">
        {{ $t("common.revert") }}
      </NButton>
      <NButton
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
import { create } from "@bufbuild/protobuf";
import { isEqual } from "lodash-es";
import { NButton } from "naive-ui";
import { computed, reactive, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { LabelListEditor } from "@/components/Label/";
import { pushNotification, useDatabaseV1Store } from "@/store";
import { type ComposedDatabase } from "@/types";
import { UpdateDatabaseRequestSchema } from "@/types/proto-es/v1/database_service_pb";
import { convertKVListToLabels, convertLabelsToKVList } from "@/utils";

type LocalState = {
  kvList: { key: string; value: string }[];
  isUpdating: boolean;
};

const props = defineProps<{
  database: ComposedDatabase;
  allowEdit: boolean;
}>();

const { t } = useI18n();
const labelListEditorRef = ref<InstanceType<typeof LabelListEditor>>();
const state = reactive<LocalState>({
  kvList: [],
  isUpdating: false,
});

const convert = () => {
  return convertLabelsToKVList(props.database.labels, true /* sort */);
};

const dirty = computed(() => {
  const original = convert();
  const local = state.kvList;
  return !isEqual(original, local);
});

const allowSave = computed(() => {
  if (!dirty.value) return false;
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
    // Won't omit empty `tenant` value.
    // Otherwise the server API won't update the labels field correctly.
    const labels = convertKVListToLabels(state.kvList, false /* !omitEmpty */);

    await useDatabaseV1Store().updateDatabase(
      create(UpdateDatabaseRequestSchema, {
        database: {
          ...props.database,
          labels,
        },
        updateMask: { paths: ["labels"] },
      })
    );
    pushNotification({
      module: "bytebase",
      style: "SUCCESS",
      title: t("common.updated"),
    });
  } finally {
    state.isUpdating = false;
  }
};

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
</script>
