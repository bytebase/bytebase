import { DataSourceType } from "../types";

const directive = {
  beforeMount(el: HTMLElement) {
    const dataSourceType = el.innerText as DataSourceType;
    switch (dataSourceType) {
      case "RO":
        el.innerText = "Read Only";
        return;
      case "RW":
        el.innerText = "Read & Write";
        return;
    }
  },
};

export default directive;
