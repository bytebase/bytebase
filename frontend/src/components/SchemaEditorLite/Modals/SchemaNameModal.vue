<template>
  <BBModal
    :title="$t('schema-editor.actions.create-schema')"
    class="shadow-inner outline outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("schema-editor.schema.name") }}</p>
      <NInput
        ref="inputRef"
        v-model:value="state.schemaName"
        class="my-2"
        :autofocus="true"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 space-x-3">
      <NButton @click="dismissModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="handleConfirmButtonClick">
        {{ $t("common.create") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { NInput } from "naive-ui";
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { useNotificationStore } from "@/store";
import { ComposedDatabase } from "@/types";
import {
  DatabaseMetadata,
  SchemaMetadata,
} from "@/types/proto/v1/database_service";
import { useSchemaEditorContext } from "../context";

const schemaNameFieldRegexp = /^\S+$/;

interface LocalState {
  schemaName: string;
}

const props = defineProps<{
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const context = useSchemaEditorContext();
const { addTab, markEditStatus } = context;
const notificationStore = useNotificationStore();
const state = reactive<LocalState>({
  schemaName: "",
});

const handleConfirmButtonClick = async () => {
  if (!schemaNameFieldRegexp.test(state.schemaName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-schema-name"),
    });
    return;
  }

  const schema = SchemaMetadata.fromPartial({
    name: state.schemaName,
  });
  /* eslint-disable-next-line vue/no-mutating-props */
  props.metadata.schemas.push(schema);
  markEditStatus(
    props.database,
    {
      database: props.metadata,
      schema,
    },
    "created"
  );
  requestAnimationFrame(() => {
    addTab({
      type: "database",
      database: props.database,
      metadata: {
        database: props.metadata,
      },
      selectedSchema: schema.name,
    });
  });

  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
