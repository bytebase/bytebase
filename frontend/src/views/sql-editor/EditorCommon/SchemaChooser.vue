<template>
  <NPopselect
    v-if="show"
    :value="chosenSchema"
    :options="options"
    :show-checkmark="true"
    :virtual-scroll="true"
    :style="overridePopoverPaneStyle"
    placement="bottom-end"
    trigger="click"
    class="schema-chooser-pane"
    @update:show="handleToggleShow"
    @update:value="handleSelect"
  >
    <template #header>
      <div class="font-medium">
        {{ $t("database.schema.select") }}
      </div>
    </template>
    <NButton
      ref="buttonRef"
      size="small"
      ghost
      type="primary"
      style="
        display: inline-flex;
        justify-content: end;
        overflow: hidden;
        --n-padding: 0 7px 0 5px;
        --n-icon-margin: 6px 2px 6px 0;
        --n-color-hover: rgb(var(--color-accent) / 0.05);
        --n-color-pressed: rgb(var(--color-accent) / 0.05);
        --n-color-focus: rgb(var(--color-accent) / 0.05);
      "
      :style="{
        maxWidth: isChosen ? '12rem' : 'unset',
      }"
    >
      <template #icon>
        <SchemaIcon
          class="w-4 h-4"
          :class="isChosen ? 'text-main' : 'text-control-placeholder'"
        />
      </template>
      <span v-if="isChosen" class="truncate text-main">
        {{ chosenSchema || $t("db.schema.default") }}
      </span>
      <span v-else class="text-control-placeholder whitespace-nowrap">
        {{ $t("database.schema.select") }}
      </span>
    </NButton>
  </NPopselect>
</template>

<script setup lang="ts">
import { useElementBounding } from "@vueuse/core";
import { NButton, NPopselect, type SelectOption } from "naive-ui";
import { storeToRefs } from "pinia";
import { computed, nextTick, ref, watch } from "vue";
import { useI18n } from "vue-i18n";
import { SchemaIcon } from "@/components/Icon";
import {
  useConnectionOfCurrentSQLEditorTab,
  useDBSchemaV1Store,
  useSQLEditorTabStore,
} from "@/store";
import { DatabaseMetadataView } from "@/types/proto/v1/database_service";
import { instanceAllowsSchemaScopedQuery } from "@/utils";

const SchemaOptionValueUnspecified = "-1";

const { t } = useI18n();
const { currentTab: tab } = storeToRefs(useSQLEditorTabStore());
const { database, instance } = useConnectionOfCurrentSQLEditorTab();
const show = computed(() => {
  return instanceAllowsSchemaScopedQuery(instance.value.engine);
});

const popoverPaneRef = ref<HTMLDivElement>();
const popoverPaneBounding = useElementBounding(popoverPaneRef);
const popoverPaneDimensions = ref({
  minWidth: 0,
  maxHeight: 0,
});

const overridePopoverPaneStyle = computed(() => {
  const style: Record<string, any> = {};
  const { maxHeight, minWidth } = popoverPaneDimensions.value;
  if (maxHeight > 0) {
    style["--n-height"] = `${maxHeight}px`;
  }
  if (minWidth > 0) {
    style["min-width"] = `${minWidth}px`;
  }
  style["maxWidth"] = "20rem";
  return style;
});

const databaseMetadata = computed(() => {
  return useDBSchemaV1Store().getDatabaseMetadata(
    database.value.name,
    DatabaseMetadataView.DATABASE_METADATA_VIEW_FULL
  );
});
const options = computed(() => {
  const options = databaseMetadata.value.schemas.map<SelectOption>(
    (schema) => ({
      value: schema.name,
      label: schema.name || t("db.schema.default"),
    })
  );
  options.unshift({
    value: SchemaOptionValueUnspecified,
    label: t("database.schema.unspecified"),
  });
  return options;
});

const chosenSchema = computed<string>({
  get() {
    const schema = tab.value?.connection.schema;
    if (schema === undefined) return SchemaOptionValueUnspecified;
    return schema;
  },
  set(value) {
    if (!tab.value) return;
    tab.value.connection.schema =
      value === SchemaOptionValueUnspecified ? undefined : value;
  },
});

const isChosen = computed(() => {
  return chosenSchema.value !== SchemaOptionValueUnspecified;
});

const handleSelect = async (value: string) => {
  chosenSchema.value = value;
};

const handleToggleShow = async (show: boolean) => {
  if (!show) return;
  await nextTick();

  const pane = document.querySelector(".schema-chooser-pane") as HTMLDivElement;
  popoverPaneRef.value = pane;
};

// Calculate the max-height of the pane
// to prevent it to be too long to overflow the bottom of the screen
watch(popoverPaneBounding.top, (top) => {
  if (!top) {
    // Cannot calculate max-height
    popoverPaneDimensions.value.maxHeight = 0;
    return;
  }
  const safeZone = 20;
  const MAX_HEIGHT = 320;
  popoverPaneDimensions.value.maxHeight = Math.min(
    window.innerHeight - top - safeZone,
    MAX_HEIGHT
  );
});

// Calculate the width of the pane
// to prevent its height varies when scrolling
watch(popoverPaneBounding.width, (width) => {
  if (!width) return;
  if (width > popoverPaneDimensions.value.minWidth) {
    popoverPaneDimensions.value.minWidth = width;
  }
});
</script>
