import { Factory } from "miragejs";
import faker from "faker";
import { UNKNOWN_ID } from "../../types";

export default {
  instanceNew: Factory.extend({
    creatorId() {
      return UNKNOWN_ID;
    },
    environmentId() {
      return UNKNOWN_ID;
    },
    name(i) {
      return "instance new" + (i + i);
    },
    externalLink() {
      return "google.com";
    },
    host(i) {
      if (i == 0) {
        return "localhost";
      } else if (i == 1) {
        return "127.0.0.1";
      } else if (i == 2) {
        return "13.24.32.122";
      } else {
        return "mydb.com";
      }
    },
    port(i) {
      if (i == 0) {
        return "3306";
      } else if (i == 1) {
        return "";
      } else if (i == 2) {
        return "15202";
      } else {
        return "5432";
      }
    },
  }),
};
