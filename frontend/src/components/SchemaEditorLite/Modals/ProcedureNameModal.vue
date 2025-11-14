<template>
  <BBModal
    :title="
      mode === 'create'
        ? $t('schema-editor.actions.create-procedure')
        : $t('schema-editor.actions.rename')
    "
    class="shadow-inner outline-solid outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("common.name") }}</p>
      <NInput
        ref="inputRef"
        v-model:value="state.procedureName"
        class="my-2"
        :autofocus="true"
      />
    </div>
    <div class="w-full flex items-center justify-end mt-2 gap-x-2">
      <NButton @click="dismissModal">
        {{ $t("common.cancel") }}
      </NButton>
      <NButton type="primary" @click="handleConfirmButtonClick">
        {{ mode === "create" ? $t("common.create") : $t("common.save") }}
      </NButton>
    </div>
  </BBModal>
</template>

<script lang="ts" setup>
import { create } from "@bufbuild/protobuf";
import type { InputInst } from "naive-ui";
import { NButton, NInput } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { useI18n } from "vue-i18n";
import { BBModal } from "@/bbkit";
import { useNotificationStore } from "@/store";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  ProcedureMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { ProcedureMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

// Procedure name must start with a non-space character, end with a non-space character.
const procedureNameFieldRegexp = /^\S\S*\S?$/;

interface LocalState {
  procedureName: string;
}

const props = defineProps<{
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  schema: SchemaMetadata;
  procedure?: ProcedureMetadata;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const { events, addTab, markEditStatus } = useSchemaEditorContext();
const inputRef = ref<InputInst>();
const notificationStore = useNotificationStore();
const mode = computed(() => {
  return props.procedure ? "edit" : "create";
});
const state = reactive<LocalState>({
  procedureName: props.procedure?.name ?? "",
});

const handleConfirmButtonClick = async () => {
  if (!procedureNameFieldRegexp.test(state.procedureName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-procedure-name"),
    });
    return;
  }
  const { schema } = props;
  const existed = schema.procedures.find(
    (procedure) => procedure.name === state.procedureName
  );
  if (existed) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-procedure-name"),
    });
    return;
  }

  if (!props.procedure) {
    const procedure = create(ProcedureMetadataSchema, {
      name: state.procedureName,
      definition: [
        "CREATE PROCEDURE `" + state.procedureName + "`(...)",
        "BEGIN",
        "  ...",
        "END",
      ].join("\n"),
    });
    schema.procedures.push(procedure);
    markEditStatus(
      props.database,
      {
        schema,
        procedure,
      },
      "created"
    );

    addTab({
      type: "procedure",
      database: props.database,
      metadata: {
        database: props.metadata,
        schema: props.schema,
        procedure,
      },
    });

    events.emit("rebuild-tree", {
      openFirstChild: false,
    });
  } else {
    const { procedure } = props;
    procedure.name = state.procedureName;
    events.emit("rebuild-edit-status", {
      resets: ["tree"],
    });
  }
  dismissModal();
};

const dismissModal = () => {
  emit("close");
};

onMounted(() => {
  inputRef.value?.focus();
});
</script>
