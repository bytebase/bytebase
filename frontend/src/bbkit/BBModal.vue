<template>
  <NModal
    :show="true"
    :auto-focus="false"
    :trap-focus="trapFocus"
    :close-on-esc="false"
    :mask-closeable="maskClosable"
    @mask-click="maskClosable && upmost && tryClose()"
  >
    <div
      v-bind="$attrs"
      class="bb-modal"
      :data-overlay-stack-id="id"
      :data-overlay-stack-upmost="upmost"
    >
      <div class="modal-header" :class="headerClass">
        <div class="text-xl text-main mr-2 flex-1 overflow-hidden">
          <slot name="title"><component :is="renderTitle" /></slot>
          <slot name="subtitle"><component :is="renderSubtitle" /></slot>
        </div>
        <button
          v-if="showClose"
          class="text-control-light"
          aria-label="close"
          @click.prevent="tryClose()"
        >
          <span class="sr-only">Close</span>
          <!-- Heroicons name: x -->
          <heroicons-solid:x class="w-6 h-6" />
        </button>
      </div>

      <div class="modal-container" :class="containerClass">
        <slot />
      </div>
    </div>
  </NModal>
</template>

<script lang="ts">
import { NModal } from "naive-ui";
import { defineComponent, h, PropType, RenderFunction } from "vue";
import { useOverlayStack } from "@/components/misc/OverlayStackManager.vue";
import { useEmitteryEventListener } from "@/composables/useEmitteryEventListener";
import type { VueClass } from "@/utils";

export default defineComponent({
  name: "BBModalV2",
  components: {
    NModal,
  },
  inheritAttrs: false,
  props: {
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
  emits: ["close"],
  setup(props, { emit }) {
    const { id, upmost, events } = useOverlayStack();

    useEmitteryEventListener(events, "esc", (e) => {
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

<style scoped lang="postcss">
.bb-modal {
  @apply bg-white shadow-lg rounded-lg pt-4 pb-4 flex pointer-events-auto;
  @apply flex-col;

  max-width: calc(100vw - 80px);
  max-height: calc(100vh - 80px);
}

.modal-header {
  @apply relative mx-8 pb-2 flex items-start justify-between border-b border-block-border;
}

.modal-container {
  @apply px-8 pt-2 max-h-screen overflow-auto w-full;
}
</style>
