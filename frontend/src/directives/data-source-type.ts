import { DataSourceType, DataSourceTypes } from "../types";

const directive = {
  beforeMount(el: HTMLElement) {
    const dataSourceType = el.innerText as DataSourceType;
    switch (dataSourceType) {
      case DataSourceTypes.ADMIN:
        el.innerText = "ADMIN";
        return;
      case DataSourceTypes.RW:
        el.innerText = "Read & Write";
        return;
      case DataSourceTypes.RO:
        el.innerText = "Read Only";
        return;
    }
  },
};

export default directive;
