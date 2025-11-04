<template>
  <BBModal
    :title="$t('schema-editor.actions.create-schema')"
    class="shadow-inner outline-solid outline-gray-200"
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
    <div class="w-full flex items-center justify-end mt-2 gap-x-2">
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
import { create } from "@bufbuild/protobuf";
import { NButton, NInput } from "naive-ui";
import { reactive } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { useNotificationStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import type { DatabaseMetadata } from "@/types/proto-es/v1/database_service_pb";
import { SchemaMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
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
const { events, addTab, markEditStatus } = useSchemaEditorContext();
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

  const schema = create(SchemaMetadataSchema, {
    name: state.schemaName,
  });
  /* eslint-disable-next-line vue/no-mutating-props */
  props.metadata.schemas.push(schema);
  markEditStatus(
    props.database,
    {
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
    events.emit("rebuild-tree", {
      openFirstChild: false,
    });
  });

  dismissModal();
};

const dismissModal = () => {
  emit("close");
};
</script>
