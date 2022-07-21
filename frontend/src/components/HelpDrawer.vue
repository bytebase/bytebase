<template>
  <n-drawer
    v-model:show="active"
    :width="380"
    :placement="placement"
    :on-after-leave="onClose"
  >
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
import { useLanguage } from "@/composables/useLanguage";
import { useUIStateStore } from "@/store";
import { EventType } from "@/types";

interface State {
  frontmatter: Record<string, string>;
  html: string;
}

export default defineComponent({
  components: { NDrawer, NDrawerContent },
  setup() {
    const event = inject("event") as Event;
    const helpName = ref<string>("");
    const isHelpGuide = ref<boolean>(false);
    const active = ref(false);
    const placement = ref<DrawerPlacement>("right");
    const { locale } = useLanguage();
    const uiStateStore = useUIStateStore();

    const state = reactive<State>({
      frontmatter: {},
      html: "",
    });

    const showHelp = async (name: string, isGuide?: boolean) => {
      if (name) {
        helpName.value = name;
        if (isGuide) {
          isHelpGuide.value = true;
        }
        const res = await fetch(
          `/help/${locale.value === "zh-CN" ? "zh" : "en"}/${name}.md`
        );
        const markdown = await res.text();
        const ast: Node = Markdoc.parse(markdown);
        const content = Markdoc.transform(ast) as Tag;
        content.attributes.class = "prose"; // style help content
        const html: string = Markdoc.renderers.html(content);

        state.frontmatter = ast.attributes.frontmatter
          ? (yaml.load(ast.attributes.frontmatter) as Record<string, string>)
          : {};
        state.html = DOMPurify.sanitize(html);
        activate("right");
      }
    };

    const onClose = () => {
      if (isHelpGuide.value) {
        if (!uiStateStore.getIntroStateByKey(`guide.${helpName.value}`)) {
          uiStateStore.saveIntroStateByKey({
            key: `guide.${helpName.value}`,
            newState: true,
          });
        }
      }
      // else do nth
    };

    onMounted(() => {
      event.on(EventType.EVENT_HELP, showHelp);
    });
    onUnmounted(() => {
      event.off(EventType.EVENT_HELP);
    });
    const activate = (place: DrawerPlacement) => {
      active.value = true;
      placement.value = place;
    };
    return {
      active,
      placement,
      activate,
      state,
      onClose,
    };
  },
});
</script>
