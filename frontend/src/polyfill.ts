(() => {
  if (!String.prototype.replaceAll) {
    String.prototype.replaceAll = function (
      search: string | RegExp,
      replacement: any
    ) {
      if (search instanceof RegExp) {
        if (!search.flags.includes("g")) {
          throw new TypeError("replaceAll must be called with a global RegExp");
        }
        return this.replace(search, replacement);
      } else {
        const escapedSearch = search.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
        const regex = new RegExp(escapedSearch, "g");
        return this.replace(regex, replacement);
      }
    };
  }
})();

export default null;
