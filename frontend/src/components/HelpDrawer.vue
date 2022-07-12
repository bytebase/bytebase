<template>
  <n-drawer v-model:show="active" :width="400" :placement="placement">
    <n-drawer-content
      :title="state.frontmatter.title"
      closable
      :native-scrollbar="false"
    >
      <template #default>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <div v-if="state.html" v-html="state.html"></div>
      </template>
    </n-drawer-content>
  </n-drawer>
</template>

<script lang="ts">
import {
  defineComponent,
  ref,
  inject,
  onMounted,
  onUnmounted,
  reactive,
} from "vue";
import { NDrawer, NDrawerContent, DrawerPlacement } from "naive-ui";
import Markdoc, { Node, Tag } from "@markdoc/markdoc";
import DOMPurify from "dompurify";
import yaml from "js-yaml";
import { Event } from "@/utils";

interface State {
  frontmatter: Record<string, string>;
  html: string;
}

export default defineComponent({
  components: { NDrawer, NDrawerContent },
  setup() {
    const event = inject("event") as Event;
    const helpHTMLString = ref("");

    const state = reactive<State>({
      frontmatter: {},
      html: "",
    });

    const showHelp = async (name: string) => {
      if (name) {
        const { default: md } = await import(
          `../../../public/help/${name}.md?raw`
        );
        const ast: Node = Markdoc.parse(md);
        const frontmatter = ast.attributes.frontmatter
          ? (yaml.load(ast.attributes.frontmatter) as Record<string, string>)
          : {};
        state.frontmatter = frontmatter;
        const content = Markdoc.transform(ast) as Tag;
        content.attributes.class = "prose";
        const html: string = Markdoc.renderers.html(content);
        state.html = DOMPurify.sanitize(html);
        helpHTMLString.value = DOMPurify.sanitize(html);
        activate("right");
      }
    };

    onMounted(() => {
      event.on("help", showHelp);
    });
    onUnmounted(() => {
      event.off("help");
    });
    const active = ref(false);
    const placement = ref<DrawerPlacement>("right");
    const activate = (place: DrawerPlacement) => {
      active.value = true;
      placement.value = place;
    };
    return {
      active,
      placement,
      activate,
      helpHTMLString,
      state,
    };
  },
});
</script>
