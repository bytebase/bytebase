import { Factory } from "miragejs";

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
