<template>
  <span ref="wrapper" class="tags">
    <span v-for="(value, i) in visible" :key="i" class="tag">
      {{ value }}
    </span>
    <span v-if="rest.length > 0" class="tag rest">
      <span>+ {{ rest.length }} ...</span>
    </span>
  </span>
</template>

<script lang="ts">
import { useResizeObserver, useWindowSize } from "@vueuse/core";
import {
  computed,
  defineComponent,
  nextTick,
  onMounted,
  PropType,
  ref,
  watch,
} from "vue";

export default defineComponent({
  name: "ResponsiveTags",
  props: {
    tags: {
      type: Array as PropType<string[]>,
      default: () => [],
    },
  },
  setup(props) {
    const clip = ref(0);
    const visible = computed(() => props.tags.slice(0, clip.value));
    const rest = computed(() => props.tags.slice(clip.value));
    const wrapper = ref<HTMLElement>();
    const wrapperWidth = ref(0); // available after first render
    const winWidth = useWindowSize().width;
    const adjusting = ref(false);

    type Direction = "more" | "less";

    useResizeObserver(wrapper, (entries) => {
      const elem = entries[0].target;
      wrapperWidth.value = elem.clientWidth;
    });

    function maybeLess() {
      const elem = wrapper.value!;
      // use the latest dom value
      // because size ref maybe not flushed yet
      const innerWidth = elem.scrollWidth;
      const outerWidth = elem.clientWidth;
      if (innerWidth > outerWidth) {
        if (clip.value > 0) {
          clip.value--;
          return true;
        }
      }
      return false;
    }

    function maybeMore() {
      const elem = wrapper.value!;
      // use the latest dom value
      // because size ref maybe not flushed yet
      const innerWidth = elem.scrollWidth;
      const outerWidth = elem.clientWidth;
      if (innerWidth <= outerWidth) {
        if (clip.value < props.tags.length) {
          clip.value++;
          return true;
        }
      }
      return false;
    }

    async function adjust(dir: Direction) {
      if (adjusting.value) return;
      adjusting.value = true;

      if (dir === "more") {
        while (maybeMore()) {
          await nextTick();
        }
        // the last `maybeMore` may cause overflow
        // so call `maybeLess` to check it
        maybeLess();
      } else {
        while (maybeLess()) {
          await nextTick();
        }
      }

      adjusting.value = false;
    }

    const onWidthChange = (newWidth: number, oldWidth: number) => {
      if (oldWidth === 0) return;
      const dir: Direction = newWidth > oldWidth ? "more" : "less";
      adjust(dir);
    };

    // debouncedWatch(wrapperWidth, onWidthChange, { debounce: 50 });
    // debouncedWatch(winWidth, onWidthChange, { debounce: 50 });
    watch(wrapperWidth, onWidthChange);
    watch(winWidth, onWidthChange);

    onMounted(() => {
      adjust("more");
    });

    watch(
      () => props.tags,
      () => {
        // reset when props changed
        clip.value = 0;
        nextTick(() => adjust("more"));
      }
    );

    return {
      clip,
      rest,
      visible,
      wrapper,
    };
  },
});
</script>

<style scoped lang="postcss">
.tags {
  @apply flex flex-1 gap-1 overflow-hidden max-h-full;
}
.tag {
  @apply inline-flex items-center bg-blue-100 border-blue-300 px-2 rounded whitespace-nowrap;
}
</style>
