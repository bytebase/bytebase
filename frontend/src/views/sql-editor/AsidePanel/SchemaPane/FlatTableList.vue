<template>
  <div class="flat-table-list h-full flex flex-col">
    <!-- Virtual List -->
    <NVirtualList
      v-if="pagedTables.length > 0"
      :items="pagedTables"
      :item-size="28"
      :item-resizable="false"
      :items-style="{ paddingTop: '4px', paddingBottom: '4px' }"
    >
      <template
        #default="{ item, index }: { item: FlatTableItem; index: number }"
      >
        <div>
          <div
            class="table-item px-2 py-1 hover:bg-control-bg-hover cursor-pointer flex items-center justify-between text-sm"
            :class="{
              'bg-link-hover font-medium': isSelected(item),
            }"
            @click="handleTableClick(item)"
            @dblclick="handleTableDoubleClick(item)"
            @contextmenu="handleContextMenu($event, item)"
            @mouseenter="handleMouseEnter($event, item)"
            @mouseleave="handleMouseLeave()"
          >
            <div class="flex items-center gap-1 truncate flex-1 min-w-0">
              <NIcon size="14" class="shrink-0">
                <TableIcon />
              </NIcon>
              <span class="truncate">
                <span v-if="item.schema" class="text-gray-500">
                  {{ item.schema }}.
                </span>
                <span>{{ item.metadata.name }}</span>
              </span>
            </div>
            <div
              class="flex items-center gap-2 text-xs text-gray-500 shrink-0 ml-2"
            >
              <span> {{ item.metadata.columns.length }} cols </span>
              <NButton
                v-if="item.metadata.columns.length > 0"
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
            class="pl-4 pr-2 bg-gray-50 border-l-2 border-gray-200"
          >
            <!-- Columns Section -->
            <div v-if="item.metadata.columns.length > 0" class="py-1">
              <div class="text-xs font-medium text-gray-600 mb-1">
                {{ $t("database.columns") }}
              </div>
              <div
                v-for="column in item.metadata.columns"
                :key="column.name"
                class="flex flex-wrap items-center justify-between gap-1 py-0.5 text-xs pl-2"
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

            <!-- Indexes Section -->
            <div
              v-if="item.metadata.indexes && item.metadata.indexes.length > 0"
              class="py-1 border-t border-gray-200"
            >
              <div class="text-xs font-medium text-gray-600 mb-1">
                {{ $t("database.indexes") }}
              </div>
              <div
                v-for="index in item.metadata.indexes"
                :key="index.name"
                class="flex flex-wrap items-center justify-between gap-1 py-0.5 text-xs pl-2"
              >
                <div class="flex items-center gap-1">
                  <NIcon size="12" class="text-gray-400">
                    <IndexIcon />
                  </NIcon>
                  <span>{{ index.name }}</span>
                </div>
                <span class="text-gray-500">
                  {{ index.unique ? "UNIQUE" : "" }}
                  {{ index.primary ? "PRIMARY" : "" }}
                </span>
              </div>
            </div>
          </div>

          <NButton
            v-if="
              index === pagedTables.length - 1 &&
              (pageIndex + 1) * PAGE_SIZE < filteredTables.length
            "
            size="tiny"
            quaternary
            class="w-full!"
            @click="() => pageIndex++"
          >
            {{ $t("common.load-more") }}
          </NButton>
        </div>
      </template>
    </NVirtualList>

    <NEmpty
      v-if="filteredTables.length === 0"
      class="mt-16"
      :description="search ? 'No tables found' : 'No tables in this database'"
    />
  </div>
</template>

<script setup lang="ts">
import { ChevronDownIcon, ChevronRightIcon } from "lucide-vue-next";
import { NButton, NEmpty, NIcon, NVirtualList } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import { ColumnIcon, IndexIcon, TableIcon } from "@/components/Icon";
import type {
  DatabaseMetadata,
  TableMetadata,
} from "@/types/proto-es/v1/database_service_pb";
import { findAncestor } from "@/utils";
import { useHoverStateContext } from "./HoverPanel";

interface FlatTableItem {
  key: string;
  schema: string;
  metadata: TableMetadata;
}

const PAGE_SIZE = 500;

const props = defineProps<{
  metadata: DatabaseMetadata;
  search?: string;
  database: string;
}>();

const emit = defineEmits<{
  select: [table: FlatTableItem];
  "select-all": [table: FlatTableItem];
  contextmenu: [event: MouseEvent, table: FlatTableItem];
}>();

// Use the existing hover state context
const {
  state: hoverState,
  position: hoverPosition,
  update: updateHoverState,
} = useHoverStateContext();

const expandedTables = ref(new Set<string>());
// Track selected table for flat list
const selectedTableKey = ref<string>();
const pageIndex = ref(0);

// Flatten and filter all tables into a single array
const filteredTables = computed(() => {
  const tables: FlatTableItem[] = [];
  const pattern = props.search?.trim().toLowerCase();

  if (!props.metadata?.schemas) return tables;

  for (const schema of props.metadata.schemas) {
    for (const table of schema.tables || []) {
      const item = {
        key: `${schema.name}/${table.name}`,
        schema: schema.name,
        metadata: table,
      };
      if (!pattern || item.key.toLowerCase().includes(pattern)) {
        tables.push(item);
      }
    }
  }

  return tables;
});

const pagedTables = computed(() =>
  filteredTables.value.slice(0, (pageIndex.value + 1) * PAGE_SIZE)
);

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

const handleMouseEnter = (event: MouseEvent, item: FlatTableItem) => {
  const target = {
    database: props.database,
    schema: item.schema,
    table: item.metadata.name,
  };

  // Use a small delay even when transitioning between nodes
  // to prevent excessive re-renders when moving quickly
  const delay = hoverState.value ? 150 : undefined;
  updateHoverState(target, "before", delay);

  nextTick().then(() => {
    // Find the table item element and position hover panel
    const wrapper = findAncestor(event.target as HTMLElement, ".table-item");
    if (!wrapper) {
      // Clear hover state if we can't find the wrapper
      updateHoverState(undefined, "after", 150);
      return;
    }
    const bounding = wrapper.getBoundingClientRect();
    hoverPosition.value.x = event.clientX;
    hoverPosition.value.y = bounding.bottom;
  });
};

const handleMouseLeave = () => {
  updateHoverState(undefined, "after");
};

// Focus search box on mount
watch(
  [() => props.metadata, () => props.search],
  () => {
    // Reset search when metadata changes
    pageIndex.value = 0;
    expandedTables.value.clear();
  },
  { immediate: true }
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
