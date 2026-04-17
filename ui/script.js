const form = document.getElementById("searchForm");
const q = document.getElementById("q");
const k = document.getElementById("k");
const meta = document.getElementById("meta");
const results = document.getElementById("results");

function esc(s) {
  return String(s)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

form.addEventListener("submit", async (e) => {
  e.preventDefault();
  const query = q.value.trim();
  const topK = k.value || "10";
  if (!query) return;

  results.innerHTML = "";
  meta.textContent = "Searching...";

  try {
    const resp = await fetch("/search?q=" + encodeURIComponent(query) + "&k=" + encodeURIComponent(topK));
    const text = await resp.text();
    if (!resp.ok) {
      meta.textContent = "Error: " + text;
      return;
    }

    const data = JSON.parse(text);
    meta.textContent = "About " + data.results_count + " results (" + data.duration_ms + " ms)";

    if (!data.results || data.results.length === 0) {
      results.innerHTML = "<div class='result'>No results found.</div>";
      return;
    }

    results.innerHTML = data.results.map((r) =>
      "<div class='result'>" +
        "<div class='url'>" + esc(r.url) + "</div>" +
        "<a class='title' href='" + esc(r.url) + "' target='_blank' rel='noopener noreferrer'>" + esc(r.title) + "</a>" +
      "</div>"
    ).join("");
  } catch (err) {
    meta.textContent = "Error: " + err.message;
  }
});
