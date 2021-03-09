import { Factory } from "miragejs";

export default {
  activateInfo: Factory.extend({
    email() {
      return "foo@example.com";
    },
    password() {
      return "blablabla";
    },
    name() {
      return "foo";
    },
    token() {
      return "12345";
    },
  }),
};
