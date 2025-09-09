<template>
  <div class="flat-table-list h-full flex flex-col">
    <!-- Virtual List -->
    <NVirtualList
      v-if="filteredTables.length > 0"
      :items="filteredTables"
      :item-size="28"
      :item-resizable="false"
      :items-style="{ paddingTop: '4px', paddingBottom: '4px' }"
    >
      <template #default="{ item }">
        <div>
          <div
            class="table-item px-2 py-1 hover:bg-control-bg-hover cursor-pointer flex items-center justify-between text-sm"
            :class="{
              'bg-link-hover font-medium': isSelected(item),
            }"
            @click="handleTableClick(item)"
            @dblclick="handleTableDoubleClick(item)"
            @contextmenu="handleContextMenu($event, item)"
          >
            <div class="flex items-center gap-1 truncate flex-1 min-w-0">
              <NIcon size="14" class="shrink-0">
                <TableIcon />
              </NIcon>
              <span class="truncate">
                <span v-if="item.schema" class="text-gray-500">
                  {{ item.schema }}.
                </span>
                <span>{{ item.table }}</span>
              </span>
            </div>
            <div
              class="flex items-center gap-2 text-xs text-gray-500 shrink-0 ml-2"
            >
              <span> {{ item.columns.length }} cols </span>
              <NButton
                v-if="item.columns.length > 0"
                size="tiny"
                quaternary
                circle
                @click.stop="toggleTableExpansion(item)"
              >
                <template #icon>
                  <NIcon :size="12">
                    <ChevronRightIcon v-if="!expandedTables.has(item.key)" />
                    <ChevronDownIcon v-else />
                  </NIcon>
                </template>
              </NButton>
            </div>
          </div>

          <!-- Expanded Details -->
          <div
            v-if="expandedTables.has(item.key)"
            class="pl-6 pr-2 py-1 bg-gray-50 border-l-2 border-gray-200"
          >
            <div
              v-for="column in item.columns"
              :key="column.name"
              class="flex flex-wrap items-center justify-between gap-1 py-0.5 text-xs"
            >
              <div class="flex items-center gap-1">
                <NIcon size="12" class="text-gray-400">
                  <ColumnIcon />
                </NIcon>
                <span>{{ column.name }}</span>
              </div>
              <span class="text-gray-500">{{ column.type }}</span>
            </div>
          </div>
        </div>
      </template>
    </NVirtualList>

    <NEmpty
      v-else
      class="mt-16"
      :description="search ? 'No tables found' : 'No tables in this database'"
    />
  </div>
</template>

<script setup lang="ts">
import { ChevronRightIcon, ChevronDownIcon } from "lucide-vue-next";
import { NVirtualList, NIcon, NButton, NEmpty } from "naive-ui";
import { computed, ref, watch } from "vue";
import { TableIcon, ColumnIcon } from "@/components/Icon";
import type {
  DatabaseMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";

interface FlatTableItem {
  key: string;
  schema: string;
  table: string;
  columns: Array<{
    name: string;
    type: string;
  }>;
  metadata: TableMetadata;
}

const props = defineProps<{
  metadata: DatabaseMetadata;
  search?: string;
}>();

const emit = defineEmits<{
  select: [table: FlatTableItem];
  "select-all": [table: FlatTableItem];
  contextmenu: [event: MouseEvent, table: FlatTableItem];
}>();

const expandedTables = ref(new Set<string>());
// Track selected table for flat list
const selectedTableKey = ref<string>();

// Flatten all tables into a single array
const allTables = computed(() => {
  const tables: FlatTableItem[] = [];

  if (!props.metadata?.schemas) return tables;

  for (const schema of props.metadata.schemas) {
    for (const table of schema.tables || []) {
      tables.push({
        key: `${schema.name}/${table.name}`,
        schema: schema.name,
        table: table.name,
        columns: (table.columns || []).map((col) => ({
          name: col.name,
          type: col.type,
        })),
        metadata: table,
      });
    }
  }

  return tables;
});

// Filter tables based on search
const filteredTables = computed(() => {
  const pattern = props.search?.toLowerCase();

  if (!pattern) {
    return allTables.value;
  }

  return allTables.value.filter((item) => {
    // Search in both schema and table names
    const fullName = `${item.schema}.${item.table}`.toLowerCase();
    return fullName.includes(pattern);
  });
});

const isSelected = (item: FlatTableItem) => {
  return selectedTableKey.value === item.key;
};

const handleTableClick = (item: FlatTableItem) => {
  selectedTableKey.value = item.key;
  emit("select", item);
};

const handleTableDoubleClick = (item: FlatTableItem) => {
  selectedTableKey.value = item.key;
  emit("select-all", item);
};

const handleContextMenu = (event: MouseEvent, item: FlatTableItem) => {
  event.preventDefault();
  selectedTableKey.value = item.key;
  emit("contextmenu", event, item);
};

const toggleTableExpansion = (item: FlatTableItem) => {
  if (expandedTables.value.has(item.key)) {
    expandedTables.value.delete(item.key);
  } else {
    expandedTables.value.add(item.key);
  }
};

// Focus search box on mount
watch(
  () => props.metadata,
  () => {
    // Reset search when metadata changes
    expandedTables.value.clear();
  }
);
</script>

<style scoped>
.flat-table-list :deep(.n-virtual-list) {
  --n-item-text-color: var(--n-text-color);
}

.table-item {
  transition: background-color 0.1s;
}

.table-item:hover {
  background-color: var(--n-close-color-hover);
}
</style>
