import { DataSourceType } from "../types";

const directive = {
  beforeMount(el: HTMLElement) {
    const dataSourceType = el.innerText as DataSourceType;
    if (dataSourceType == "RO") {
      el.innerText = "Read Only";
    } else if (dataSourceType == "RW") {
      el.innerText = "Read & Write";
    }
  },
};

export default directive;
