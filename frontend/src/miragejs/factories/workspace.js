import { Factory } from "miragejs";
import faker from "faker";

export default {
  workspace: Factory.extend({
    name(i) {
      return "ws " + (i + 1);
    },
    slug(i) {
      return "ws" + (i + 1);
    },
    afterCreate(workspace, server) {
      server.create("activity", {
        workspace,
        name: workspace.name + " activity1 " + faker.fake("{{lorem.word}}"),
      });
      server.create("activity", {
        workspace,
        name: workspace.name + " activity2 " + faker.fake("{{lorem.word}}"),
      });
    },
  }),
};
