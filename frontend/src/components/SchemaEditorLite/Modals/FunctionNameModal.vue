<template>
  <BBModal
    :title="
      mode === 'create'
        ? $t('schema-editor.actions.create-function')
        : $t('schema-editor.actions.rename')
    "
    class="shadow-inner outline-solid outline-gray-200"
    @close="dismissModal"
  >
    <div class="w-72">
      <p>{{ $t("common.name") }}</p>
      <NInput
        ref="inputRef"
        v-model:value="state.functionName"
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
  FunctionMetadata,
  SchemaMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { FunctionMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";

// Function name must start with a non-space character, end with a non-space character.
const functionNameFieldRegexp = /^\S\S*\S?$/;

interface LocalState {
  functionName: string;
}

const props = defineProps<{
  database: ComposedDatabase;
  metadata: DatabaseMetadata;
  schema: SchemaMetadata;
  func?: FunctionMetadata;
}>();

const emit = defineEmits<{
  (event: "close"): void;
}>();

const { t } = useI18n();
const { events, addTab, markEditStatus } = useSchemaEditorContext();
const inputRef = ref<InputInst>();
const notificationStore = useNotificationStore();
const mode = computed(() => {
  return props.func ? "edit" : "create";
});
const state = reactive<LocalState>({
  functionName: props.func?.name ?? "",
});

const handleConfirmButtonClick = async () => {
  if (!functionNameFieldRegexp.test(state.functionName)) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.invalid-function-name"),
    });
    return;
  }
  const { schema } = props;
  const existed = schema.functions.find(
    (func) => func.name === state.functionName
  );
  if (existed) {
    notificationStore.pushNotification({
      module: "bytebase",
      style: "CRITICAL",
      title: t("schema-editor.message.duplicated-function-name"),
    });
    return;
  }

  if (!props.func) {
    const func = create(FunctionMetadataSchema, {
      name: state.functionName,
      definition: [
        "CREATE FUNCTION `" + state.functionName + "`(...) RETURNS ...",
        "    DETERMINISTIC",
        "BEGIN",
        "  ...",
        "END",
      ].join("\n"),
    });
    schema.functions.push(func);
    markEditStatus(
      props.database,
      {
        schema,
        function: func,
      },
      "created"
    );

    addTab({
      type: "function",
      database: props.database,
      metadata: {
        database: props.metadata,
        schema: props.schema,
        function: func,
      },
    });

    events.emit("rebuild-tree", {
      openFirstChild: false,
    });
  } else {
    const { func } = props;
    func.name = state.functionName;
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
