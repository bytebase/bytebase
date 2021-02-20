import { Factory } from "miragejs";

export default {
  workspace: Factory.extend({
    name(i) {
      return "ws " + (i + 1);
    },
    slug(i) {
      return "ws" + (i + 1);
    },
    afterCreate(workspace, server) {},
  }),
};
