import { Factory } from "miragejs";

export default {
  databasePatch: Factory.extend({
    ownerId() {
      return "200";
    },
  }),
};
