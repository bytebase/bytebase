<template>
  <div class="divide-block-border space-y-4 h-full">
    <SearchBox
      v-model:value="state.searchText"
      style="max-width: 100%"
      :placeholder="$t('schema-template.classification.search')"
    />

    <NTree
      ref="treeRef"
      block-line
      style="height: 100%; user-select: none"
      :data="treeData"
      :multiple="false"
      :selectable="true"
      :pattern="state.searchText"
      :show-irrelevant-nodes="false"
      :expand-on-click="true"
      :render-suffix="renderSuffix"
      :render-label="renderLabel"
      :node-props="nodeProps"
      :virtual-scroll="false"
      :theme-overrides="{ nodeHeight: '21px' }"
    />
  </div>
</template>

<script lang="ts" setup>
import { NTree, TreeOption } from "naive-ui";
import { computed, reactive, h } from "vue";
import { DataClassificationSetting_DataClassificationConfig } from "@/types/proto/v1/setting_service";
import { getHighlightHTMLByKeyWords } from "@/utils";
import ClassificationLevelBadge from "./ClassificationLevelBadge.vue";

const props = defineProps<{
  classificationConfig: DataClassificationSetting_DataClassificationConfig;
}>();

const emit = defineEmits<{
  (event: "apply", classificationId: string): void;
}>();

interface LocalState {
  searchText: string;
}

interface TreeNode extends TreeOption {
  key: string;
  label: string;
  levelId?: string;
  children?: TreeNode[];
}

interface ClassificationMap {
  [key: string]: {
    id: string;
    label: string;
    levelId?: string;
    children: ClassificationMap;
  };
}

const state = reactive<LocalState>({
  searchText: "",
});

const nodeProps = ({ option }: { option: TreeOption }) => {
  return {
    onClick(e: MouseEvent) {
      if (!option.isLeaf || !option.key) {
        return;
      }
      emit("apply", option.key as string);
    },
  };
};

const sortClassification = (
  item1: { id: string },
  item2: { id: string }
): number => {
  const n1 = Number(item1.id.split("-").join(""));
  const n2 = Number(item2.id.split("-").join(""));
  return n1 - n2;
};

const treeData = computed((): TreeNode[] => {
  const classifications = Object.values(
    props.classificationConfig.classification
  ).sort(sortClassification);

  const classificationMap: ClassificationMap = {};
  for (const classification of classifications) {
    const ids = classification.id.split("-");
    let tmp = classificationMap;
    for (let i = 0; i < ids.length - 1; i++) {
      tmp = tmp[ids.slice(0, i + 1).join("-")].children;
      if (!tmp) {
        break;
      }
    }
    if (tmp) {
      tmp[classification.id] = {
        id: classification.id,
        label: classification.title,
        levelId: classification.levelId,
        children: {},
      };
    }
  }
  return getTreeNodeList(classificationMap);
});

const getTreeNodeList = (classificationMap: ClassificationMap): TreeNode[] => {
  return Object.values(classificationMap)
    .sort(sortClassification)
    .map((item) => {
      const children = getTreeNodeList(item.children);
      return {
        key: item.id,
        label: `${item.id} ${item.label}`,
        levelId: item.levelId,
        isLeaf: children.length === 0,
        children,
      };
    });
};

const renderSuffix = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  if (!node.levelId) {
    return null;
  }

  return h(ClassificationLevelBadge, {
    showText: false,
    classification: node.key,
    classificationConfig: props.classificationConfig,
  });
};

const renderLabel = ({ option }: { option: TreeOption }) => {
  const node = option as any as TreeNode;
  return h("span", {
    innerHTML: getHighlightHTMLByKeyWords(node.label, state.searchText),
  });
};
</script>
