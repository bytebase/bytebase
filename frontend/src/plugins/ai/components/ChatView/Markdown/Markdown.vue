<template>
  <AstToMarkdown :ast="mdast" class="text-sm">
    <template #code="parsedAst">
      <CodeBlock :code="parsedAst.value" v-bind="codeBlockProps" />
    </template>
    <template #inlineCode="parsedAst">
      <HighlightCodeBlock
        :code="parsedAst.value"
        class="inline-block bg-gray-200 px-0.5 mx-0.5"
      />
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
import HighlightCodeBlock from "@/components/HighlightCodeBlock.vue";
import AstToMarkdown from "./AstToVNode.vue";
import CodeBlock, { type CodeBlockProps } from "./CodeBlock.vue";

const props = defineProps<{
  content: string;
  codeBlockProps: CodeBlockProps;
}>();

const processor = unified().use(remarkParse).use(remarkGfm);

const mdast = computed(() => {
  const tree = processor.parse(props.content ?? "");
  return tree;
});
</script>
