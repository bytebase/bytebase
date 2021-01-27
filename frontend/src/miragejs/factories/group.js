import { Factory, association } from "miragejs";
import faker from "faker";

export default {
  group: Factory.extend({
    name(i) {
      return "grp " + (i + 1);
    },
    slug(i) {
      return "grp" + (i + 1);
    },
    namespace() {
      return "";
    },
    afterCreate(group, server) {},
  }),
};
