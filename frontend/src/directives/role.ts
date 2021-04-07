import { RoleType } from "../types";
import { isDBA, isDeveloper, isOwner } from "../utils";

const directive = {
  beforeMount(el: HTMLElement) {
    const role = el.innerText as RoleType;
    if (isOwner("OWNER") || isOwner("DBA") || isOwner("DEVELOPER")) {
      el.innerText = "Owner";
    } else if (isDBA("OWNER") || isDBA("DBA") || isDBA("DEVELOPER")) {
      el.innerText = "DBA";
    } else if (
      isDeveloper("OWNER") ||
      isDeveloper("DBA") ||
      isDeveloper("DEVELOPER")
    ) {
      el.innerText = "Developer";
    }
  },
};

export default directive;
