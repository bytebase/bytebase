import { RoleType } from "../types";

const directive = {
  beforeMount(el: HTMLElement) {
    const role = el.innerText as RoleType;
    if (role == "OWNER") {
      el.innerText = "Owner";
    } else if (role == "DBA") {
      el.innerText = "DBA";
    } else if (role == "DEVELOPER") {
      el.innerText = "Developer";
    }
  },
};

export default directive;
