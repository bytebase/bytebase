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
          <NRadio :value="'LIST'" :label="'Existing Schema Design'" />
          <NRadio :value="'CREATE'" :label="'New Schema Design'" />
        </NRadioGroup>
      </div>

      <template v-if="state.tab === 'LIST' && !state.selectedSchemaDesign">
        <SchemaDesignTable
          v-if="ready"
          :schema-designs="schemaDesignList"
          @click="handleSchemaDesignClick"
        />
        <div v-else class="w-full h-[20rem] flex items-center justify-center">
          <BBSpin />
        </div>
      </template>
      <template v-else>
        <div class="w-full flex flex-row justify-start items-center">
          <span class="flex w-40 items-center">{{ $t("common.name") }}</span>
          <BBTextField
            class="w-60 !py-1.5"
            :value="state.schemaDesignName"
            @input="
              state.schemaDesignName = ($event.target as HTMLInputElement).value
            "
          />
        </div>
        <BaselineSchemaSelector
          v-if="isCreating"
          :baseline-schema="state.baselineSchema"
          @update="handleBaselineSchemaChange"
        />
        <template v-if="state.selectedSchemaDesign">
          <SchemaDesigner
            ref="schemaDesignerRef"
            :key="schemaDesignId"
            :engine="state.selectedSchemaDesign.engine"
            :schema-design="state.selectedSchemaDesign"
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
import {
  ChangeHistory,
  DatabaseMetadata,
} from "@/types/proto/v1/database_service";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { useDatabaseV1Store, useProjectV1ByUID } from "@/store";
import {
  useSchemaDesignList,
  useSchemaDesignStore,
} from "@/store/modules/schemaDesign";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import BaselineSchemaSelector from "../BaselineSchemaSelector.vue";
import SchemaDesigner from "../index.vue";
import { databaseNamePrefix } from "@/store/modules/v1/common";
import { watch } from "vue";
import { mergeSchemaEditToMetadata } from "../common/util";

interface BaselineSchema {
  // The uid of project.
  projectId?: string;
  // The uid of database.
  databaseId?: string;
  changeHistory?: ChangeHistory;
}

interface LocalState {
  tab: "LIST" | "CREATE";
  schemaDesignName: string;
  baselineSchema: BaselineSchema;
  selectedSchemaDesign?: SchemaDesign;
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
const schemaDesignStore = useSchemaDesignStore();
const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  tab: "LIST",
  schemaDesignName: "",
  baselineSchema: {},
});
const isCreating = computed(() => state.tab === "CREATE");

watch(
  () => state.tab,
  () => {
    state.selectedSchemaDesign = undefined;
  }
);

const prepareSchemaDesign = async () => {
  const changeHistory = state.baselineSchema.changeHistory;
  if (changeHistory && state.baselineSchema.databaseId) {
    const database = databaseStore.getDatabaseByUID(
      state.baselineSchema.databaseId
    );
    const baselineMetadata = await schemaDesignStore.parseSchemaString(
      changeHistory.schema,
      database.instanceEntity.engine
    );
    return SchemaDesign.fromPartial({
      engine: database.instanceEntity.engine,
      baselineSchema: changeHistory.schema,
      baselineSchemaMetadata: baselineMetadata,
      schema: changeHistory.schema,
      schemaMetadata: baselineMetadata,
    });
  }
  return undefined;
};

const schemaDesignId = computed(() => {
  if (!state.selectedSchemaDesign || !state.selectedSchemaDesign.name) {
    return uniqueId();
  } else {
    return state.selectedSchemaDesign.name;
  }
});

onMounted(() => {
  console.log("mounted", schemaDesignList.value);
});

const handleSchemaDesignClick = (schemaDesign: SchemaDesign) => {
  state.schemaDesignName = schemaDesign.title;
  state.selectedSchemaDesign = schemaDesign;
};

const handleBaselineSchemaChange = async (baselineSchema: BaselineSchema) => {
  state.baselineSchema = baselineSchema;

  if (isCreating.value) {
    state.selectedSchemaDesign = await prepareSchemaDesign();
  }
};

const cancel = () => {
  emit("dismiss");
};

const handleConfirm = async () => {
  if (!state.selectedSchemaDesign) {
    return;
  }

  const designerState = schemaDesignerRef.value;
  if (!designerState) {
    // Should not happen.
    throw new Error("schema designer is undefined");
  }

  if (isCreating.value) {
    if (state.schemaDesignName === "") {
      return;
    }

    const { project } = useProjectV1ByUID(state.baselineSchema.projectId || "");
    const database = useDatabaseV1Store().getDatabaseByUID(
      state.baselineSchema.databaseId || ""
    );
    const baselineDatabase = `${database.instanceEntity.name}/${databaseNamePrefix}${state.baselineSchema.databaseId}`;
    const metadata = mergeSchemaEditToMetadata(
      designerState.editableSchemas,
      state.selectedSchemaDesign.baselineSchemaMetadata ||
        DatabaseMetadata.fromPartial({})
    );

    console.log(
      project.value.name,
      designerState.editableSchemas,
      SchemaDesign.fromPartial({
        title: state.schemaDesignName,
        // Keep schema empty in frontend. Backend will generate the design schema.
        schema: "",
        schemaMetadata: metadata,
        baselineSchema: state.selectedSchemaDesign.baselineSchema,
        baselineSchemaMetadata:
          state.selectedSchemaDesign.baselineSchemaMetadata,
        engine: state.selectedSchemaDesign.engine,
        baselineDatabase: baselineDatabase,
        schemaVersion: state.baselineSchema.changeHistory?.name || "",
      })
    );

    // await schemaDesignStore.createSchemaDesign(
    //   project.value.name,
    //   SchemaDesign.fromPartial({
    //     title: state.schemaDesignName,
    //     // Keep schema empty in frontend. Backend will generate the design schema.
    //     schema: "",
    //     schemaMetadata: metadata,
    //     baselineSchema: state.selectedSchemaDesign.baselineSchema,
    //     baselineSchemaMetadata:
    //       state.selectedSchemaDesign.baselineSchemaMetadata,
    //     engine: state.selectedSchemaDesign.engine,
    //     baselineDatabase: baselineDatabase,
    //     schemaVersion: state.baselineSchema.changeHistory?.name || "",
    //   })
    // );
  } else {
    // do patch schema design
  }
};
</script>
