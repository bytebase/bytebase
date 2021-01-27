import { Factory } from "miragejs";
import faker from "faker";

export default {
  loginInfo: Factory.extend({
    usename() {
      return "foo@example.com";
    },
    password() {
      return "blablabla";
    },
  }),
};
