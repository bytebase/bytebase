import { Factory } from "miragejs";
import faker from "faker";

export default {
  repository: Factory.extend({
    type() {
      return "Gitlab";
    },
    content() {
      return {
        accessToken: faker.internet.password(10, false, /[0-9A-Za-z]/),
        url: "localhost:1234",
      };
    },
  }),
};
