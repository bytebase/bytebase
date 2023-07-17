<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{ $t("database.design-schema") }}</span>
      </div>
    </template>

    <div
      class="space-y-3 w-[calc(100vw-8rem)] lg:w-[60rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div>
        <NRadioGroup v-model:value="state.tab">
          <NRadio
            :key="'LIST'"
            :value="'LIST'"
            :label="'Existing Schema Design'"
          />
          <NRadio :key="'VIEW'" :value="'VIEW'" :label="'New Schema Design'" />
        </NRadioGroup>
      </div>

      <template v-if="state.tab === 'LIST'">
        <SchemaDesignTable v-if="ready" :schema-designs="schemaDesignList" />
        <div v-else class="w-full h-[20rem] flex items-center justify-center">
          <BBSpin />
        </div>
      </template>
      <template v-else>
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center">Name</span>
          <BBTextField class="w-60 !py-1.5" :value="state.schemaDesignName" />
        </div>
        <BaselineSchemaSelector @update="handleBaselineSchemaChange" />
        <template v-if="schemaDesign">
          <SchemaDesigner
            ref="schemaDesignerRef"
            :key="schemaDesignId"
            :engine="schemaDesign.engine"
            :schema-design="schemaDesign"
          />
        </template>
      </template>
    </div>

    <template #footer>
      <div class="flex-1 flex items-center justify-between">
        <div></div>

        <div class="flex items-center justify-end gap-x-3">
          <NButton @click.prevent="cancel">
            {{ $t("common.cancel") }}
          </NButton>
          <NButton type="primary" @click.prevent="handleConfirm">
            {{ $t("common.next") }}
          </NButton>
        </div>
      </div>
    </template>
  </DrawerContent>
</template>

<script lang="ts" setup>
import { uniqueId } from "lodash-es";
import { NRadioGroup, NRadio } from "naive-ui";
import { computed, onMounted, reactive, ref } from "vue";
import { ChangeHistory } from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { useDBSchemaV1Store, useDatabaseV1Store } from "@/store";
import {
  useSchemaDesignList,
  useSchemaDesignStore,
} from "@/store/modules/schemaDesign";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import BaselineSchemaSelector from "../BaselineSchemaSelector.vue";
import SchemaDesigner from "../index.vue";

interface BaselineSchema {
  projectId?: string;
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  tab: "LIST" | "VIEW";
  schemaDesignName: string;
  baselineSchema: BaselineSchema;
}

defineProps({
  projectId: {
    type: String,
    default: undefined,
  },
});

const emit = defineEmits(["dismiss"]);
const schemaDesignerRef = ref<InstanceType<typeof SchemaDesigner>>();

const databaseStore = useDatabaseV1Store();
const dbSchemaStore = useDBSchemaV1Store();
const schemaDesignStore = useSchemaDesignStore();
const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  tab: "LIST",
  schemaDesignName: "",
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

const schemaDesignId = computed(() => {
  if (!schemaDesign.value || !schemaDesign.value.name) {
    return uniqueId();
  } else {
    return schemaDesign.value.name;
  }
});

onMounted(() => {
  console.log("mounted", schemaDesignList.value);
});

const handleBaselineSchemaChange = (baselineSchema: BaselineSchema) => {
  state.baselineSchema = baselineSchema;
};

const cancel = () => {
  emit("dismiss");
};

const handleConfirm = () => {
  if (state.tab === "VIEW") {
    if (!schemaDesign.value) {
      return;
    }

    const schemaDesigner = schemaDesignerRef.value;
    if (!schemaDesigner) {
      // Should not happen.
      throw new Error("schemaDesigner is undefined");
    }

    const isCreating = schemaDesign.value.name === "";
    if (isCreating) {
      if (state.schemaDesignName === "") {
        console.log("schemaDesignName is empty");
        return;
      }

      schemaDesignStore.createSchemaDesign(
        state.baselineSchema.projectId || "",
        SchemaDesign.fromPartial({
          title: state.schemaDesignName,
          // Keep schema empty in frontend. Backend will generate the design schema.
          schema: "",
          // TODO(steven): calculate design schema metadata with metadata and editableSchemas.
          schemaMetadata: schemaDesigner.metadata,
          baselineSchema: state.baselineSchema.changeHistory?.schema || "",
          baselineSchemaMetadata: schemaDesigner.baselineMetadata,
          engine: schemaDesign.value.engine,
          baselineDatabase: state.baselineSchema.databaseId || "",
          schemaVersion: state.baselineSchema.changeHistory?.name || "",
        })
      );
    } else {
      // do patch schema design
    }
  }
};
</script>
