import { Factory } from "miragejs";
import { UNKNOWN_ID } from "../../types";

export default {
  instance: Factory.extend({
    rowStatus(i) {
      return i > 3 ? "ARCHIVED" : "NORMAL";
    },
    creatorId() {
      return UNKNOWN_ID;
    },
    updaterId() {
      return UNKNOWN_ID;
    },
    createdTs(i) {
      return Date.now() - (i + 1) * 1800 * 1000;
    },
    lastUpdatedTs(i) {
      return Date.now() - i * 3600 * 1000;
    },
    name(i) {
      return "instance " + (i + i);
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
