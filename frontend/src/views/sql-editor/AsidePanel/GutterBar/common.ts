import type { ButtonProps } from "naive-ui";
import { computed, type MaybeRef, unref } from "vue";

export type Size = "small" | "medium" | "large";

export const useButton = (options: {
  size: MaybeRef<Size>;
  active: MaybeRef<boolean>;
  disabled: MaybeRef<boolean>;
}) => {
  const props = computed(() => {
    const props: ButtonProps = {
      tag: "div",
      disabled: unref(options.disabled),
      size: unref(options.size),
    };
    if (unref(options.active)) {
      props.secondary = true;
      props.type = "primary";
    } else {
      props.quaternary = true;
      props.type = "default";
    }
    props.disabled = unref(options.disabled);

    return props;
  });
  const style = computed(() => {
    const parts: string[] = [];
    const size = unref(options.size);

    if (size === "small") {
      parts.push("--n-height: 32px");
      parts.push("--n-padding: 4px 8px");
      parts.push("--n-icon-size: 16px");
    } else if (size === "medium") {
      parts.push("--n-height: 40px");
      parts.push("--n-padding: 6px 10px");
      parts.push("--n-icon-size: 20px");
    } else if (size === "large") {
      parts.push("--n-height: 48px");
      parts.push("--n-padding: 12px 12px");
      parts.push("--n-icon-size: 24px");
    }
    return parts.join("; ");
  });
  return { props, style };
};
