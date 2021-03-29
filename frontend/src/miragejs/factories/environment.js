import { Factory } from "miragejs";

export default {
  environment: Factory.extend({
    name(i) {
      if (i == 0) {
        return "Sandbox A";
      } else if (i == 1) {
        return "Integration";
      } else if (i == 2) {
        return "Staging";
      } else {
        if (i == 3) {
          return "Prod";
        }
        return "Prod " + (i - 2);
      }
    },
    order(i) {
      return i;
    },
  }),
};
