import { Factory } from "miragejs";
import faker from "faker";

export default {
  project: Factory.extend({
    name() {
      return "project " + faker.fake("{{lorem.word}}");
    },
    slug(i) {
      return "pro" + (i + 1);
    },
    namespace() {
      return "";
    },
    lastUpdated() {
      return Date.now() - 3600 * 1000;
    },
    pinned(i) {
      return i % 2 == 0;
    },
    afterCreate(project, server) {
      server.create("repository", { project });
    },
  }),
};
