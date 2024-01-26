document.addEventListener("DOMContentLoaded", () => {
  const populateBodySize = () => {
    const rect = document.body.getBoundingClientRect();
    window.parent.postMessage(
      {
        source: "bb.markdown.renderer",
        key: window.name,
        width: rect.width,
        height: rect.height,
      },
      "*"
    );
  };

  const ob = new ResizeObserver((entries) => {
    const bodyEntry = entries.find((entry) => entry.target === document.body);
    if (!bodyEntry) return;
    populateBodySize();
  });

  ob.observe(document.body);
});
