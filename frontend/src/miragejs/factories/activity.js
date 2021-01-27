import { randomNumber } from "./utils";
import { Factory } from "miragejs";
import faker from "faker";

export default {
  activity: Factory.extend({
    name() {
      return "activity " + faker.fake("{{lorem.word}}");
    },
    description() {
      return faker.fake("{{lorem.sentence}}");
    },
    creator() {
      return (
        faker.fake("{{name.lastName}}") + " " + faker.fake("{{name.firstName}}")
      );
    },
    link(i) {
      return "actlink" + (i + 1);
    },
    startTs() {
      return Date.now() - 3600 * 1000;
    },
    endTs() {
      return Date.now() + randomNumber(3600 * 1000);
    },
  }),
};
