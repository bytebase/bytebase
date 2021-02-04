import { Factory } from "miragejs";

export default {
  batchUpdate: Factory.extend({
    idList() {
      return ["0", "1"];
    },
    fieldMaskList() {
      return ["age", "name"];
    },
    rowValueList() {
      return [
        ["10", "Bob"],
        ["25", "Alice"],
      ];
    },
  }),
};
