<template>
  <AstToMarkdown :ast="mdast" class="text-sm leading-6">
    <template #code="parsedAst">
      <CodeBlock :code="parsedAst.value" />
    </template>
    <template #inlineCode="parsedAst">
      <code>{{ parsedAst.value }}</code>
    </template>
    <template #image="parsedAst">
      <img :src="parsedAst.url" />
    </template>
  </AstToMarkdown>
</template>

<script lang="ts" setup>
import remarkGfm from "remark-gfm";
import remarkParse from "remark-parse";
import { unified } from "unified";
import { computed } from "vue";
import AstToMarkdown from "./AstToVNode.vue";
import CodeBlock from "./CodeBlock.vue";

const props = defineProps<{
  content: string;
}>();

const processor = unified().use(remarkParse).use(remarkGfm);

const mdast = computed(() => {
  const tree = processor.parse(props.content ?? "");
  return tree;
});
</script>
