import { computed } from "vue";
import { useRoute, useRouter } from "vue-router";

export const useRouteHash = <TS extends readonly string[]>(
  defaultValue: TS[number],
  values?: TS,
  method: "replace" | "push" = "replace"
) => {
  type T = TS[number];

  const route = useRoute();
  const router = useRouter();
  const normalize = (value: T): T => {
    if (!values) return value;
    if (values.includes(value)) return value;
    return defaultValue;
  };
  return computed<T>({
    get() {
      const hash = (route.hash ?? "").replace(/^#?/, "") as T;
      return normalize(hash || defaultValue);
    },
    set(value) {
      value = normalize(value);
      router[method](`#${value}`);
    },
  });
};
