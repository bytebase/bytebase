import { useVirtualizer } from "@tanstack/vue-virtual";
import hljs from "highlight.js/lib/core";
import { computed, defineComponent, h, onMounted, ref, watch } from "vue";

// Enable virtual scrolling for code with more lines than this
const VIRTUAL_SCROLL_THRESHOLD = 100;

export default defineComponent({
  name: "HighlightCodeBlock",
  props: {
    code: {
      type: String,
      required: true,
    },
    language: {
      type: String,
      default: "sql",
    },
    tag: {
      type: String,
      default: "pre",
    },
    lazy: {
      type: Boolean,
      default: false,
    },
    virtual: {
      type: Boolean,
      default: false,
    },
    lineHeight: {
      type: Number,
      default: 20,
    },
  },
  setup(props) {
    const highlighted = ref(!props.lazy);
    const highlightedCode = ref("");
    const containerRef = ref<HTMLElement>();
    const lineCache = ref<Map<number, string>>(new Map());

    const lines = computed(() => props.code.split("\n"));
    const shouldUseVirtual = computed(
      () => props.virtual && lines.value.length > VIRTUAL_SCROLL_THRESHOLD
    );

    const highlightLine = (line: string): string => {
      try {
        const result = hljs.highlight(line, {
          language: props.language,
          ignoreIllegals: true,
        });
        return result.value;
      } catch {
        return line;
      }
    };

    const doHighlight = () => {
      if (shouldUseVirtual.value) {
        // For virtual mode, highlighting happens per-line on demand
        highlighted.value = true;
        return;
      }

      const highlightFn = () => {
        const result = hljs.highlight(props.code, {
          language: props.language,
        });
        highlightedCode.value = result.value;
        highlighted.value = true;
      };

      if (props.lazy) {
        // Async highlighting
        if (typeof requestIdleCallback !== "undefined") {
          requestIdleCallback(highlightFn);
        } else {
          setTimeout(highlightFn, 0);
        }
      } else {
        // Sync highlighting
        highlightFn();
      }
    };

    // Virtual scrolling setup
    const virtualizer = useVirtualizer(
      computed(() => ({
        count: lines.value.length,
        getScrollElement: () => containerRef.value ?? null,
        estimateSize: () => props.lineHeight,
        overscan: 10,
      }))
    );

    const virtualItems = computed(() => virtualizer.value.getVirtualItems());

    // Initialize highlighting
    if (!props.lazy) {
      doHighlight();
    } else {
      onMounted(() => {
        doHighlight();
      });
    }

    watch(
      () => props.code,
      () => {
        highlighted.value = false;
        lineCache.value.clear();
        doHighlight();
      }
    );

    return {
      highlighted,
      highlightedCode,
      containerRef,
      lines,
      shouldUseVirtual,
      virtualizer,
      virtualItems,
      highlightLine,
      lineCache,
    };
  },
  render() {
    const { code, language, tag } = this.$props;
    const { class: additionalClass } = this.$attrs;

    // Virtual scrolling mode
    if (this.shouldUseVirtual && this.highlighted) {
      const virtualHeight = this.virtualizer.getTotalSize();
      const virtualStart = this.virtualItems[0]?.start ?? 0;

      return h(
        "div",
        {
          ref: "containerRef",
          class: ["overflow-auto", additionalClass],
        },
        [
          h(
            "div",
            {
              style: {
                height: `${virtualHeight}px`,
                position: "relative",
              },
            },
            [
              h(
                tag,
                {
                  class: [language],
                  style: {
                    position: "absolute",
                    top: 0,
                    left: 0,
                    width: "100%",
                    transform: `translateY(${virtualStart}px)`,
                    margin: 0,
                  },
                },
                this.virtualItems.map((virtualRow) => {
                  const lineIndex = virtualRow.index;
                  const line = this.lines[lineIndex];

                  // Check cache first
                  if (!this.lineCache.has(lineIndex)) {
                    this.lineCache.set(lineIndex, this.highlightLine(line));
                  }

                  return h("div", {
                    key: String(virtualRow.key),
                    style: {
                      height: `${virtualRow.size}px`,
                    },
                    innerHTML: this.lineCache.get(lineIndex),
                  });
                })
              ),
            ]
          ),
        ]
      );
    }

    // Non-virtual mode - show plain text while highlighting
    if (!this.highlighted) {
      return h(
        tag,
        {
          class: [language, additionalClass],
        },
        code
      );
    }

    // Highlighted content
    return h(tag, {
      class: [language, additionalClass],
      innerHTML: this.highlightedCode,
    });
  },
});
