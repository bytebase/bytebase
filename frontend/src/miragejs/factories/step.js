import { Factory } from "miragejs";
import faker from "faker";

export default {
  step: Factory.extend({
    name() {
      return "step " + faker.fake("{{lorem.word}}");
    },
    slug(i) {
      return "step" + i;
    },
    lastUpdated() {
      return Date.now() - 3600 * 1000;
    },
    created() {
      return Date.now();
    },
    status() {
      let dice = Math.random();
      return dice < 0.33 ? "RUNNING" : dice < 0.66 ? "SUCCEEDED" : "FAILED";
    },
  }),
};
