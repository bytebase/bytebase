<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{ $t("database.design-schema") }}</span>
      </div>
    </template>

    <div
      class="space-y-4 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div v-if="ready">
        <SchemaDesignTable :schema-designs="schemaDesignList" />
        <hr />

        <BaselineSchemaSelector @update="handleBaselineSchemaChange" />

        <template v-if="schemaDesign">
          <SchemaDesigner
            :engine="schemaDesign.engine"
            :schema-design="schemaDesign"
          />
        </template>
      </div>
      <div v-else class="w-full h-[20rem] flex items-center justify-center">
        <BBSpin />
      </div>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div></div>

        <div class="flex items-center justify-end gap-x-3">
          <NButton @click.prevent="cancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton type="primary" @click.prevent="">
            {{ $t("common.next") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { computed, onMounted, reactive } from "vue";
import { useSchemaDesignList } from "@/store/modules/schemaDesign";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import BaselineSchemaSelector from "../BaselineSchemaSelector.vue";
import SchemaDesigner from "../index.vue";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";

interface BaselineSchema {
  projectId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  tab: "TABLE" | "VIEW";
  baselineSchema: BaselineSchema;
}

defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const emit = defineEmits(["dismiss"]);

const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  tab: "TABLE",
  baselineSchema: {},
});

const schemaDesign = computed(() => {
  const changeHistory = state.baselineSchema.changeHistory;
  if (changeHistory && state.baselineSchema.databaseId) {
    const database = databaseStore.getDatabaseByUID(
      state.baselineSchema.databaseId
    );
    const metadata = dbSchemaStore.getDatabaseMetadata(database.name);
    return SchemaDesign.fromPartial({
      engine: database.instanceEntity.engine,
      baselineSchema: changeHistory.schema,
      // TODO: parse schema to metadata.
      baselineSchemaMetadata: metadata,
      schema: changeHistory.schema,
      schemaMetadata: metadata,
    });
  }
  return undefined;
});

onMounted(() => {
  console.log("mounted", schemaDesignList.value);
});

const cancel = () => {
  emit("dismiss");
};

const handleBaselineSchemaChange = (baselineSchema: BaselineSchema) => {
  state.baselineSchema = baselineSchema;
  console.log("state.baselineSchema", state.baselineSchema);
};
</script>
