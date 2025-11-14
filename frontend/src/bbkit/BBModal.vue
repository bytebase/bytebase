<template>
  <NModal
    :show="show"
    :auto-focus="false"
    :trap-focus="trapFocus"
    :close-on-esc="false"
    :mask-closeable="maskClosable"
    @mask-click="maskClosable && upmost && tryClose()"
  >
    <div
      v-bind="$attrs"
      class="bg-white shadow-lg rounded-md py-3 flex pointer-events-auto flex-col gap-3"
      :style="{
        maxWidth: 'calc(100vw - 80px)',
        maxHeight: 'calc(100vh - 80px)',
      }"
      :data-overlay-stack-id="id"
      :data-overlay-stack-upmost="upmost"
    >
      <div
        class="relative mx-4 pb-2 flex items-center justify-between border-b border-block-border"
        :class="headerClass"
      >
        <div class="text-lg text-main mr-2 flex-1 overflow-hidden">
          <slot name="title"><component :is="renderTitle" /></slot>
          <slot name="subtitle"><component :is="renderSubtitle" /></slot>
        </div>
        <NButton
          v-if="showClose"
          quaternary
          size="small"
          aria-label="close"
          @click.prevent="tryClose()"
        >
          <span class="sr-only">Close</span>
          <XIcon class="w-5 h-auto hover:opacity-80" />
        </NButton>
      </div>

      <div
        class="px-4 max-h-screen overflow-auto w-full h-full"
        :class="containerClass"
      >
        <slot />
      </div>
    </div>
  </NModal>
</template>

<script lang="ts">
import { XIcon } from "lucide-vue-next";
import { NButton, NModal } from "naive-ui";
import type { PropType, RenderFunction } from "vue";
import { defineComponent, h, toRef } from "vue";
import { useOverlayStack } from "@/components/misc/OverlayStackManager.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import type { VueClass } from "@/utils";

export default defineComponent({
  name: "BBModalV2",
  components: {
    NModal,
    NButton,
    XIcon,
  },
  inheritAttrs: false,
  props: {
    show: {
      type: Boolean,
      default: true,
    },
    title: {
      default: "",
      type: [String, Function] as PropType<string | RenderFunction>,
    },
    subtitle: {
      default: "",
      type: [String, Function] as PropType<string | RenderFunction>,
    },
    showClose: {
      type: Boolean,
      default: true,
    },
    headerClass: {
      type: [String, Object, Array] as PropType<VueClass>,
      default: undefined,
    },
    containerClass: {
      type: [String, Object, Array] as PropType<VueClass>,
      default: undefined,
    },
    closeOnEsc: {
      type: Boolean,
      default: true,
    },
    maskClosable: {
      type: Boolean,
      // Default to `false` to make it behaves consistent with legacy BBModal
      default: false,
    },
    beforeClose: {
      type: Function as PropType<() => Promise<boolean>>,
      default: undefined,
    },
    trapFocus: {
      type: Boolean,
      default: undefined,
    },
  },
  emits: ["close", "update:show"],
  setup(props, { emit }) {
    const { id, upmost, events } = useOverlayStack(toRef(props, "show"));

    useEmitteryEventListener(events, "esc", () => {
      if (upmost.value && props.closeOnEsc) {
        tryClose();
      }
    });

    const tryClose = async () => {
      const { beforeClose } = props;
      if (beforeClose) {
        const pass = await beforeClose();
        if (!pass) return;
      }
      emit("close");
      emit("update:show", false);
    };

    const renderTitle = () => {
      if (typeof props.title === "function") {
        return props.title();
      }
      return props.title;
    };

    const renderSubtitle = () => {
      if (typeof props.subtitle === "function") {
        return props.subtitle();
      }
      if (props.subtitle) {
        return h(
          "div",
          {
            class: "text-sm text-control whitespace-nowrap",
          },
          [h("span", { class: "inline-block" }, props.subtitle)]
        );
      }
      return null;
    };

    return {
      tryClose,
      renderTitle,
      renderSubtitle,
      id,
      upmost,
    };
  },
});
</script>
