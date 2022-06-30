import { reactive, useAttrs } from "vue";

const dataLabelRegex = RegExp("data-.+");

const useDataLabelAttrs = (prefix = "", endfix = "") => {
  const attrs = useAttrs();
  const dataLabelAttrs = reactive<any>({});

  for (const key in attrs) {
    if (dataLabelRegex.test(key)) {
      dataLabelAttrs[key] = prefix + attrs[key] + endfix;
    }
  }

  return dataLabelAttrs;
};

export default useDataLabelAttrs;
