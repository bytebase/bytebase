<template>
  <Drawer @close="$emit('dismiss')">
    <DrawerContent
      :title="$t('deployment-config.matched-databases.all')"
      :content-props="{
        bodyContentClass: 'relative',
      }"
    >
      <div class="focus:outline-none w-auto" tabindex="0">
        <div class="grid grid-cols-3 gap-3 border rounded-lg p-4">
          <DatabaseMatrixItem
            v-for="item in databases"
            :key="item.name"
            :database="item"
          />
        </div>
      </div>
    </DrawerContent>
  </Drawer>
</template>

<script lang="ts" setup>
import type { PropType } from "vue";
import { Drawer, DrawerContent } from "@/components/v2";
import type { ComposedDatabase } from "@/types";
import DatabaseMatrixItem from "./DatabaseMatrixItem.vue";

defineProps({
  databases: {
    type: Object as PropType<ComposedDatabase[]>,
    required: true,
  },
});

defineEmits(["dismiss"]);
</script>
