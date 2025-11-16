<template>
  <div class="flex flex-col gap-y-2">
    <div>
      <p>{{ $t("sql-editor.choose-folder") }}</p>
      <span class="textinfolabel">
        {{ $t("sql-editor.choose-folder-tips") }}
      </span>
    </div>
    <NPopover
      placement="bottom"
      :show="showPopover"
      :show-arrow="false"
      trigger="manual"
      :width="folderInputRef?.wrapperElRef?.clientWidth"
    >
      <template #trigger>
        <NInput
          ref="folderInputRef"
          :value="formattedFolderPath.split('/').join(' / ')"
          :placeholder="$t('sql-editor.choose-folder')"
          @focus="onFocus"
          @update:value="onInput"
        />
      </template>
      <NTree
        ref="folderTreeRef"
        block-line
        block-node
        virtual-scroll
        :clearable="false"
        :filterable="true"
        :pattern="folderPath"
        :checkable="false"
        :check-on-click="true"
        :selectable="true"
        :selected-keys="[folderPath]"
        :multiple="false"
        :data="folderTree.children"
        :render-prefix="renderPrefix"
        :expanded-keys="expandedKeysArray"
        @update:expanded-keys="(keys: string[]) => expandedKeys = new Set(keys)"
        @update:selected-keys="onSelect"
      />
    </NPopover>
  </div>
</template>

<script lang="tsx" setup>
import { onClickOutside } from "@vueuse/core";
import { NInput, NPopover, NTree, type TreeOption } from "naive-ui";
import { computed, nextTick, ref, watch } from "vue";
import TreeNodePrefix from "@/views/sql-editor/AsidePanel/WorksheetPane/SheetList/TreeNodePrefix.vue";
import type { WorksheetFolderNode } from "@/views/sql-editor/Sheet";
import { useSheetContextByView } from "@/views/sql-editor/Sheet";

const props = defineProps<{
  folder: string;
}>();

const emits = defineEmits<{
  (event: "update:folder", folder: string): void;
}>();

const folderPath = ref<string>("");

watch(
  () => props.folder,
  (val) => (folderPath.value = val),
  { immediate: true }
);

watch(
  () => folderPath.value,
  (val) => emits("update:folder", val)
);

const { folderTree, folderContext } = useSheetContextByView("my");

const expandedKeys = ref<Set<string>>(new Set([]));
const expandedKeysArray = computed(() => Array.from(expandedKeys.value));
const folderInputRef = ref<InstanceType<typeof NInput>>();
const folderTreeRef = ref<InstanceType<typeof NTree>>();
const showPopover = ref<boolean>(false);

onClickOutside(folderTreeRef, () => {
  showPopover.value = false;
});

const formattedFolderPath = computed(() => {
  let val = folderPath.value.replace(folderContext.rootPath.value, "");
  if (val[0] === "/") {
    val = val.slice(1);
  }
  return val;
});

const onFocus = () => {
  showPopover.value = true;
};

const onSelect = (keys: string[]) => {
  folderPath.value = keys[0] ?? "";
  nextTick(() => (showPopover.value = false));
};

const onInput = (val: string) => {
  let changedVal = val;
  if (val.endsWith(" /")) {
    changedVal = val.slice(0, -2);
  }
  const rawPath = changedVal
    .split("/")
    .map((p) => p.trim())
    .join("/");
  folderPath.value = rawPath;
};

const renderPrefix = ({ option }: { option: TreeOption }) => {
  const node = option as WorksheetFolderNode;
  return (
    <TreeNodePrefix
      node={node}
      expandedKeys={expandedKeys.value}
      rootPath={folderContext.rootPath.value}
      view={"my"}
    />
  );
};

defineExpose({
  folderPath: formattedFolderPath,
  folders: computed(() =>
    formattedFolderPath.value
      .split("/")
      .map((p) => p.trim())
      .filter((p) => p)
  ),
});
</script>
