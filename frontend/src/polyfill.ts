(() => {
  if (!String.prototype.replaceAll) {
    String.prototype.replaceAll = function (search: any, replacement: any) {
      return this.replace(new RegExp(search, "g"), replacement);
    };
  }
})();

export default null;
