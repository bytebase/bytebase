import { Factory } from "miragejs";

export default {
  loginInfo: Factory.extend({
    email() {
      return "foo@example.com";
    },
    password() {
      return "blablabla";
    },
  }),
};
