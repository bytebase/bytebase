<template>
  <DrawerContent>
    <template #header>
      <div class="flex flex-col gap-y-1">
        <span>{{ $t("schema-designer.quick-action") }}</span>
      </div>
    </template>

    <div
      class="space-y-3 w-[calc(100vw-24rem)] min-w-[64rem] max-w-[calc(100vw-8rem)] overflow-x-auto"
    >
      <div
        class="w-full border-b pb-2 mb-2 flex flex-row justify-between items-center"
      >
        <div class="flex flex-row justify-start items-center space-x-2"></div>
        <div>
          <NButton type="primary" @click="state.showCreatePanel = true">
            <heroicons-solid:plus class="w-4 h-auto mr-0.5" />
            <span>{{ $t("schema-designer.new-design") }}</span>
          </NButton>
        </div>
      </div>

      <SchemaDesignTable
        v-if="ready"
        :schema-designs="schemaDesignList"
        @click="handleSchemaDesignItemClick"
      />
      <div v-else class="w-full h-[20rem] flex items-center justify-center">
        <BBSpin />
      </div>
    </div>
  </DrawerContent>

  <CreateSchemaDesignPanel
    v-if="state.showCreatePanel"
    @dismiss="state.showCreatePanel = false"
    @created="
      (schemaDesign) => {
        state.showCreatePanel = false;
        handleSchemaDesignItemClick(schemaDesign);
      }
    "
  />

  <EditSchemaDesignPanel
    v-if="state.selectedSchemaDesignName"
    :schema-design-name="state.selectedSchemaDesignName"
    @dismiss="state.selectedSchemaDesignName = undefined"
  />
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui";
import { reactive } from "vue";
import { SchemaDesign } from "@/types/proto/v1/schema_design_service";
import { useSchemaDesignList } from "@/store/modules/schemaDesign";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import { DrawerContent } from "@/components/v2";
import CreateSchemaDesignPanel from "../CreateSchemaDesignPanel.vue";
import EditSchemaDesignPanel from "../EditSchemaDesignPanel.vue";

interface LocalState {
  showCreatePanel: boolean;
  selectedSchemaDesignName?: string;
}

const { schemaDesignList, ready } = useSchemaDesignList();
const state = reactive<LocalState>({
  showCreatePanel: false,
});

const handleSchemaDesignItemClick = async (schemaDesign: SchemaDesign) => {
  state.selectedSchemaDesignName = schemaDesign.name;
};
</script>
