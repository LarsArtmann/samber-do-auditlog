// E2E smoke for the samber-do-auditlog live dashboard (TODO 2026-09-11 §b.1).
// Asserts: zero console errors, signals initialize, all fragments render,
// tab switching works (datastar expression evaluation under CSP).
import puppeteer from "puppeteer-core";

const URL = process.env.DASH_URL || "http://localhost:7777/debug/di/";
const exe = process.env.CHROME;
if (!exe) throw new Error("CHROME env var required");

const browser = await puppeteer.launch({
	executablePath: exe,
	args: ["--no-sandbox", "--disable-gpu", "--disable-dev-shm-usage"],
});
const page = await browser.newPage();
const consoleErrors = [];
page.on("console", (msg) => {
	if (msg.type() === "error") consoleErrors.push(`console.error: ${msg.text()}`);
});
page.on("pageerror", (err) => consoleErrors.push(`pageerror: ${err.message}`));

await page.goto(URL, { waitUntil: "domcontentloaded", timeout: 20000 });
await page.waitForSelector("#services-tbody tr", { timeout: 20000 });
await new Promise((r) => setTimeout(r, 3000));

const snap = await page.evaluate(() => {
	const txt = (id) => (document.getElementById(id)?.textContent || "").trim();
	const count = (sel) => document.querySelectorAll(sel).length;
	return {
		serviceRows: count("#services-tbody tr"),
		eventRows: count("#events-tbody tr"),
		stats: txt("stats"),
		legendItems: count("#legend *"),
		scopeTree: (document.getElementById("scope-tree-container")?.innerHTML || "").length,
		graphContent: (document.getElementById("graph-container")?.innerHTML || "").length,
		connection: txt("connection-status"),
		containerId: txt("container-id"),
		tabs: count(".tab[data-tab]"),
		timelineItems: count("#tab-timeline *"),
	};
});

// Exercise datastar expression evaluation: click every tab, verify it activates.
const tabResults = {};
for (const name of ["services", "scopes", "graph", "timeline", "events"]) {
	await page.click(`#tab-btn-${name}`).catch(() => {});
	await new Promise((r) => setTimeout(r, 400));
	tabResults[name] = await page.evaluate(
		(n) => document.getElementById(`tab-btn-${n}`)?.classList.contains("active"),
		name,
	);
}

await page.screenshot({ path: "/tmp/dash-e2e/services.png" });

await browser.close();

const checks = {
	"service rows rendered": snap.serviceRows > 0,
	"event rows rendered": snap.eventRows > 0,
	"stats populated": snap.stats.length > 0,
	"legend rendered": snap.legendItems > 0,
	"scope tree rendered": snap.scopeTree > 50,
	"graph rendered": snap.graphContent > 50,
	"connection indicator": snap.connection.length > 0,
	"container id shown": snap.containerId.length > 0,
	"5 tabs present": snap.tabs === 5,
	"timeline rendered": snap.timelineItems > 0,
	"all tabs activate via clicks": Object.values(tabResults).every(Boolean),
	"zero console errors": consoleErrors.length === 0,
};
console.log(JSON.stringify({ snap, tabResults, consoleErrors }, null, 2));
let failed = 0;
for (const [name, ok] of Object.entries(checks)) {
	console.log(`${ok ? "PASS" : "FAIL"}  ${name}`);
	if (!ok) failed++;
}
process.exit(failed ? 1 : 0);
