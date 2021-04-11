import { DatabaseSyncStatus } from "../types";

const directive = {
  beforeMount(el: HTMLElement) {
    const syncStatus = el.innerText as DatabaseSyncStatus;
    switch (syncStatus) {
      case "OK":
        el.innerText = "OK";
        return;
      case "DRIFTED":
        el.innerText = "Drifted";
        return;
      case "NOT_FOUND":
        el.innerText = "Not found";
        return;
    }
  },
};

export default directive;
