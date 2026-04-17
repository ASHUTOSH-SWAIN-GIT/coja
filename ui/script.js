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
    const terms = (data.terms || []).join(", ");
    meta.textContent = "Results: " + data.results_count + " | Terms: [" + terms + "] | " + data.duration_ms + "ms";

    if (!data.results || data.results.length === 0) {
      results.innerHTML = "<div class='result'>No results found.</div>";
      return;
    }

    results.innerHTML = data.results.map((r, i) =>
      "<div class='result'>" +
        "<div class='title'>" + (i + 1) + ". " + esc(r.Title) + "</div>" +
        "<div class='sub'>doc=" + r.DocID + " | score=" + Number(r.Score).toFixed(4) + "</div>" +
      "</div>"
    ).join("");
  } catch (err) {
    meta.textContent = "Error: " + err.message;
  }
});
