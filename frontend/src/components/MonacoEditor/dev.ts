import { useLocalStorage } from "@vueuse/core";
import { defineComponent, h } from "vue";
import { BBSwitch } from "@/bbkit";

export const SupportedLSPTypes = ["OLD", "NEW"] as const;
export type LSPType = typeof SupportedLSPTypes[number];

export const StoredLSPType = useLocalStorage<LSPType>(
  "bb.sql-editor.dev-lsp-type",
  "OLD",
  {
    serializer: {
      read(raw: LSPType) {
        if (!SupportedLSPTypes.includes(raw)) return "OLD";
        return raw;
      },
      write(value) {
        return value;
      },
    },
  }
);

export const LSPTypeSwitch = defineComponent({
  name: "LSPTypeSwitch",
  render() {
    const label = h("span", {}, "[DEV] LSP Type");
    const switcher = h(
      BBSwitch,
      {
        text: true,
        value: StoredLSPType.value === "NEW",
        onToggle(on: boolean) {
          StoredLSPType.value = on ? "NEW" : "OLD";
          location.reload();
        },
      },
      {
        on: () => h("span", { class: "font-medium scale-x-75" }, "NEW"),
        off: () => h("span", { class: "font-medium scale-x-75" }, "OLD"),
      }
    );
    return h(
      "div",
      {
        class: "flex flex-row items-center gap-x-2 justify-between",
        onClick: (e: MouseEvent) => {
          e.stopPropagation();
          e.preventDefault();
        },
      },
      [label, switcher]
    );
  },
});
