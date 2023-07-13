<template>
  <div>
    <div class="w-full"></div>
    <SchemaDesignTable :schema-designs="schemaDesignList" />
  </div>
</template>

<script lang="ts" setup>
import { reactive } from "vue";
import SchemaDesignTable from "./SchemaDesignTable.vue";
import { useSchemaDesignList } from "@/store/modules/schemaDesign";

interface LocalState {
  tab: "TABLE" | "VIEW";
}

const { schemaDesignList } = useSchemaDesignList();
const state = reactive<LocalState>({
  tab: "TABLE",
});
</script>
