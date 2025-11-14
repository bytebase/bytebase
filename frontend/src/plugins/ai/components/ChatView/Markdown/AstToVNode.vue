<template>
  <AstToMarkdown />
</template>

<script setup lang="ts">
import type { Root } from "mdast";
import { type CustomRender, mdastToVNode, type State } from "./utils";

const props = defineProps<{
  ast: Root;
}>();

const slots = defineSlots<CustomRender>();

function AstToMarkdown() {
  const state: State = {
    slots,
    definitionById: new Map(),
  };
  const result = mdastToVNode[props.ast.type](props.ast, state);

  return result;
}
</script>
