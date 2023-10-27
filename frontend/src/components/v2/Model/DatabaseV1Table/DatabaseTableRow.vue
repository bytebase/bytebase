<template>
  <div
    v-if="showSelectionColumn"
    class="bb-grid-cell !px-2"
    @click.stop.prevent
  >
    <slot name="selection" :database="database" />
  </div>
  <div class="bb-grid-cell">
    <div class="flex items-center space-x-2">
      <SQLEditorButtonV1
        :database="database"
        :disabled="!allowQuery"
        :tooltip="true"
        @failed="$emit('goto-sql-editor-failed')"
      />
      <DatabaseV1Name :database="database" :link="false" tag="span" />
      <BBBadge
        v-if="isPITRDatabaseV1(database)"
        text="PITR"
        :can-remove="false"
        class="text-xs"
      />
    </div>
  </div>
  <div v-if="showEnvironmentColumn" class="bb-grid-cell">
    <EnvironmentV1Name
      :environment="environment ?? database.effectiveEnvironmentEntity"
      :link="false"
      tag="div"
    />
  </div>
  <div v-if="showSchemaVersionColumn" class="hidden lg:bb-grid-cell">
    {{ database.schemaVersion }}
  </div>
  <div v-if="showProjectColumn" class="bb-grid-cell">
    <ProjectCol
      :project="database.projectEntity"
      :mode="mode"
      :show-tenant-icon="showTenantIcon"
    />
  </div>
  <div v-if="showInstanceColumn" class="bb-grid-cell">
    <InstanceV1Name
      :instance="database.instanceEntity"
      :link="false"
      tag="div"
    />
  </div>
  <div v-if="showLabelsColumn" class="bb-grid-cell !py-1">
    <LabelsColumn :labels="database.labels" :show-count="1" placeholder="-" />
  </div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import { InstanceV1Name, EnvironmentV1Name } from "@/components/v2";
import { useEnvironmentV1Store } from "@/store";
import { ComposedDatabase } from "@/types";
import { isPITRDatabaseV1 } from "@/utils";
import LabelsColumn from "./LabelsColumn.vue";
import ProjectCol from "./ProjectCol.vue";

const props = defineProps<{
  database: ComposedDatabase;
  mode: string;
  showSelectionColumn: boolean;
  showMiscColumn: boolean;
  showSchemaVersionColumn: boolean;
  showProjectColumn: boolean;
  showTenantIcon: boolean;
  showEnvironmentColumn: boolean;
  showInstanceColumn: boolean;
  showLabelsColumn: boolean;
  allowQuery: boolean;
}>();

defineEmits(["goto-sql-editor-failed"]);

const environment = computed(() => {
  return useEnvironmentV1Store().getEnvironmentByName(
    props.database.environment
  );
});
</script>
