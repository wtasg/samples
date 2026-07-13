document.addEventListener("DOMContentLoaded", () => {
    checkConnection();
    refreshSchema();
});

// Templates definitions
const templates = {
    create: `db.createCollection("employees");\n`,
    insert: `db.employees.insert({"_id": "e1", "name": "Alice", "dept": "Engineering", "salary": 95000});\ndb.employees.insert({"_id": "e2", "name": "Bob", "dept": "Engineering", "salary": 88000});\ndb.employees.insert({"_id": "e3", "name": "Carol", "dept": "Design", "salary": 82000});\ndb.employees.insert({"_id": "e4", "name": "Dave", "dept": "Engineering", "salary": 91000});`,
    find: `db.employees.find({});`,
    filter: `db.employees.find({"dept": {"$prefix": "Eng"}, "salary": {"$gt": 90000}});`,
    sort: `db.employees.find({}).sort({"salary": -1});`
};

let currentDocs = [];
let currentViewMode = 'json'; // 'json' or 'table'

function loadTemplate(type) {
    const editor = document.getElementById("command-editor");
    editor.value = templates[type];
    editor.focus();
}

async function checkConnection() {
    const dot = document.querySelector(".status-dot");
    const text = document.querySelector(".status-text");

    try {
        const r = await fetch("/api/ping");
        const data = await r.json();
        if (data.online) {
            dot.className = "status-dot online";
            text.textContent = `Online: ${data.version}`;
        } else {
            dot.className = "status-dot offline";
            text.textContent = `Offline: ${data.error || 'Server error'}`;
        }
    } catch (e) {
        dot.className = "status-dot offline";
        text.textContent = "Offline: Connection Refused";
    }
}

async function refreshSchema() {
    const list = document.getElementById("collection-list");
    try {
        const r = await fetch("/api/collections");
        if (!r.ok) throw new Error("failed to fetch collections");
        const collections = await r.json();

        if (collections.length === 0) {
            list.innerHTML = `<div class="empty-state">No collections found.</div>`;
            return;
        }

        list.innerHTML = collections.map(c => `
            <div class="collection-item" onclick="loadCollectionFind('${c.name}')">
                <div class="collection-name">${c.name}</div>
                <div class="collection-stats">
                    <span>${c.doc_count} document${c.doc_count === 1 ? '' : 's'}</span>
                    <span>${formatBytes(c.size_bytes)}</span>
                </div>
            </div>
        `).join('');

    } catch (e) {
        list.innerHTML = `<div class="empty-state" style="color: var(--status-red)">Error loading schema</div>`;
    }
}

function loadCollectionFind(name) {
    const editor = document.getElementById("command-editor");
    editor.value = `db.${name}.find({});`;
    editor.focus();
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

async function runQuery() {
    const command = document.getElementById("command-editor").value.trim();
    const btn = document.getElementById("btn-run");
    const time = document.getElementById("exec-time");
    const banner = document.getElementById("status-message");
    const jsonContainer = document.getElementById("json-container");
    const tableContainer = document.getElementById("table-container");
    const viewToggle = document.getElementById("view-toggle");

    if (!command) return;

    btn.disabled = true;
    time.textContent = "Executing...";
    
    // Clear previous view
    banner.style.display = "none";
    jsonContainer.style.display = "none";
    tableContainer.style.display = "none";
    viewToggle.style.display = "none";

    const startTime = performance.now();

    try {
        const r = await fetch("/api/query", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ command })
        });
        if (!r.ok) throw new Error(`HTTP error ${r.status}`);
        const result = await r.json();

        const duration = (performance.now() - startTime).toFixed(1);
        time.textContent = `Executed in ${duration}ms`;

        if (!result.ok) {
            banner.className = "message-banner error";
            banner.textContent = result.error;
            banner.style.display = "block";
        } else {
            if (result.docs && result.docs.length > 0) {
                currentDocs = result.docs;
                viewToggle.style.display = "flex";
                renderResults();
            } else {
                banner.className = "message-banner success";
                banner.textContent = result.message || "Command executed successfully.";
                banner.style.display = "block";
            }
            refreshSchema();
            checkConnection();
        }

    } catch (e) {
        time.textContent = "";
        banner.className = "message-banner error";
        banner.textContent = `Failed to contact server: ${e.message}`;
        banner.style.display = "block";
    } finally {
        btn.disabled = false;
    }
}

function setViewMode(mode) {
    currentViewMode = mode;
    document.getElementById("btn-json-view").className = mode === 'json' ? "toggle-btn active" : "toggle-btn";
    document.getElementById("btn-table-view").className = mode === 'table' ? "toggle-btn active" : "toggle-btn";
    renderResults();
}

function renderResults() {
    const jsonContainer = document.getElementById("json-container");
    const tableContainer = document.getElementById("table-container");

    if (currentViewMode === 'json') {
        tableContainer.style.display = "none";
        document.getElementById("json-output").textContent = JSON.stringify(currentDocs, null, 2);
        jsonContainer.style.display = "block";
    } else {
        jsonContainer.style.display = "none";
        renderTable(currentDocs);
        tableContainer.style.display = "block";
    }
}

function renderTable(docs) {
    const header = document.getElementById("table-header");
    const body = document.getElementById("table-body");

    // Extract unique field names across all returned documents
    const colSet = new Set();
    // Always put _id first if exists
    colSet.add("_id");
    docs.forEach(doc => {
        Object.keys(doc).forEach(key => colSet.add(key));
    });
    const cols = Array.from(colSet);

    header.innerHTML = cols.map(c => `<th>${c}</th>`).join('');

    body.innerHTML = docs.map(doc => {
        const cells = cols.map(c => {
            const val = doc[c];
            let displayVal = '';
            if (val === null || val === undefined) {
                displayVal = '<span style="color: var(--text-secondary)">null</span>';
            } else if (typeof val === 'object') {
                displayVal = JSON.stringify(val);
            } else {
                displayVal = val;
            }
            return `<td>${displayVal}</td>`;
        }).join('');
        return `<tr>${cells}</tr>`;
    }).join('');
}
