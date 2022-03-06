import { DataSourceType } from "../types";

const directive = {
  beforeMount(el: HTMLElement) {
    const dataSourceType = el.innerText as DataSourceType;
    switch (dataSourceType) {
      case "ADMIN":
        el.innerText = "ADMIN";
        return;
      case "RW":
        el.innerText = "Read & Write";
        return;
      case "RO":
        el.innerText = "Read Only";
        return;
    }
  },
};

export default directive;
