<template>
  <NTabs v-model:value="state.selectedTab" type="line">
    <NTabPane
      name="FIELD_TEMPLATE"
      :tab="$t('schema-template.field-template.self')"
    >
      <FieldTemplates :show-engine-filter="true" />
    </NTabPane>
    <NTabPane
      name="COLUMN_TYPE_RESTRICTION"
      :tab="$t('schema-template.column-type-restriction.self')"
    >
      <ColumnTypes />
    </NTabPane>
  </NTabs>
</template>

<script lang="ts" setup>
import { NTabs, NTabPane } from "naive-ui";
import { reactive, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import ColumnTypes from "@/views/SchemaTemplate/ColumnTypes.vue";
import FieldTemplates from "@/views/SchemaTemplate/FieldTemplates.vue";

interface LocalState {
  selectedTab: "FIELD_TEMPLATE" | "COLUMN_TYPE_RESTRICTION";
}

const route = useRoute();
const router = useRouter();
const state = reactive<LocalState>({
  selectedTab: "FIELD_TEMPLATE",
});

watch(
  () => route.hash,
  () => {
    if (route.hash === "#column-type-restriction") {
      state.selectedTab = "COLUMN_TYPE_RESTRICTION";
    } else {
      state.selectedTab = "FIELD_TEMPLATE";
    }
  },
  {
    immediate: true,
  }
);

watch(
  () => state.selectedTab,
  () => {
    if (state.selectedTab === "COLUMN_TYPE_RESTRICTION") {
      router.push({ hash: "#column-type-restriction" });
    } else if (state.selectedTab === "FIELD_TEMPLATE") {
      router.push({ hash: "" });
    }
  }
);
</script>
