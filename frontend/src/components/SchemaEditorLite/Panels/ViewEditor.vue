<template>
  <CommonCodeEditor
    :db="db"
    :code="view.definition"
    :readonly="disallowChangeView"
    :status="status"
    @update:code="handleUpdateDefinition"
  >
    <template #preview>
      <PreviewPane
        :db="db"
        :database="database"
        :schema="schema"
        :mocked="mocked"
        :title="$t('schema-editor.preview-view-definition')"
      />
    </template>
  </CommonCodeEditor>
</template>

<script setup lang="ts">
import { create } from "@bufbuild/protobuf";
import { cloneDeep } from "lodash-es";
import { computed } from "vue";
import type { ComposedDatabase } from "@/types";
import type {
  DatabaseMetadata,
  SchemaMetadata,
  ViewMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { DatabaseMetadataSchema } from "@/types/proto-es/v1/database_service_pb";
import { useSchemaEditorContext } from "../context";
import type { EditStatus } from "../types";
import CommonCodeEditor from "./CommonCodeEditor.vue";
import PreviewPane from "./PreviewPane.vue";

const props = defineProps<{
  db: ComposedDatabase;
  database: DatabaseMetadata;
  schema: SchemaMetadata;
  view: ViewMetadata;
}>();

const { readonly, markEditStatus, getSchemaStatus, getViewStatus } =
  useSchemaEditorContext();

const statusForSchema = () => {
  return getSchemaStatus(props.db, {
    schema: props.schema,
  });
};
const status = computed(() => {
  return getViewStatus(props.db, {
    schema: props.schema,
    view: props.view,
  });
});
const markStatus = (status: EditStatus) => {
  markEditStatus(
    props.db,
    {
      schema: props.schema,
      view: props.view,
    },
    status
  );
};

const disallowChangeView = computed(() => {
  if (readonly.value) {
    return true;
  }
  return statusForSchema() === "dropped" || status.value === "dropped";
});

const mocked = computed(() => {
  const { database, schema, view } = props;

  const mockedView = cloneDeep(view);
  const mockedDatabase = create(DatabaseMetadataSchema, {
    name: database.name,
    characterSet: database.characterSet,
    collation: database.collation,
    schemas: [
      {
        name: schema.name,
        views: [mockedView],
      },
    ],
  });

  return { metadata: mockedDatabase };
});

const handleUpdateDefinition = (code: string) => {
  // eslint-disable-next-line vue/no-mutating-props
  props.view.definition = code;

  if (status.value === "normal") {
    markStatus("updated");
  }
};
</script>
