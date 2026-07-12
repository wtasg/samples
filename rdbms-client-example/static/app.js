document.addEventListener("DOMContentLoaded", () => {
    checkConnection();
    refreshSchema();
});

// Templates definitions
const templates = {
    create: "CREATE TABLE employees (id INT, name TEXT, department TEXT, salary INT);\n",
    insert: "INSERT INTO employees VALUES (1, 'Alice', 'Engineering', 95000);\nINSERT INTO employees VALUES (2, 'Bob', 'Engineering', 88000);\nINSERT INTO employees VALUES (3, 'Carol', 'Design', 82000);\nINSERT INTO employees VALUES (4, 'Dave', 'Engineering', 91000);",
    select: "SELECT * FROM employees;",
    filter: "SELECT * FROM employees WHERE department LIKE 'Eng%' AND salary > 90000;",
    orderby: "SELECT * FROM employees ORDER BY salary DESC;"
};

function loadTemplate(type) {
    const editor = document.getElementById("sql-editor");
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
    const list = document.getElementById("table-list");
    try {
        const r = await fetch("/api/tables");
        if (!r.ok) throw new Error("failed to fetch tables");
        const tables = await r.json();

        if (tables.length === 0) {
            list.innerHTML = `<div class="empty-state">No tables found.</div>`;
            return;
        }

        list.innerHTML = tables.map(t => {
            const colsHtml = t.columns.map(c => `
                <div class="column-item">
                    <span>${c.name}</span>
                    <span>${c.type}${c.pk ? ' <span class="pk-marker">PK</span>' : ''}</span>
                </div>
            `).join('');

            return `
                <div class="table-item" onclick="loadTableSelect('${t.name}')">
                    <div class="table-name">${t.name}</div>
                    <div class="column-list">${colsHtml}</div>
                </div>
            `;
        }).join('');

    } catch (e) {
        list.innerHTML = `<div class="empty-state" style="color: var(--status-red)">Error loading schema</div>`;
    }
}

function loadTableSelect(name) {
    const editor = document.getElementById("sql-editor");
    editor.value = `SELECT * FROM ${name};`;
    editor.focus();
}

async function runQuery() {
    const sql = document.getElementById("sql-editor").value.trim();
    const btn = document.getElementById("btn-run");
    const time = document.getElementById("exec-time");
    const banner = document.getElementById("status-message");
    const tableContainer = document.getElementById("table-container");

    if (!sql) return;

    btn.disabled = true;
    time.textContent = "Executing...";
    
    // Clear previous view
    banner.style.display = "none";
    tableContainer.style.display = "none";

    const startTime = performance.now();

    try {
        const r = await fetch("/api/query", {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ sql })
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
            // Check if SELECT statement
            if (result.columns && result.columns.length > 0) {
                renderTable(result.columns, result.rows);
            } else {
                banner.className = "message-banner success";
                banner.textContent = result.message || "Query executed successfully.";
                banner.style.display = "block";
            }
            // Something changed in database (e.g. CREATE/DROP), refresh schema
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

function renderTable(cols, rows) {
    const header = document.getElementById("table-header");
    const body = document.getElementById("table-body");
    const tableContainer = document.getElementById("table-container");

    header.innerHTML = cols.map(c => `<th>${c}</th>`).join('');

    if (!rows || rows.length === 0) {
        body.innerHTML = `<tr><td colspan="${cols.length}" style="text-align: center; color: var(--text-secondary)">No rows returned</td></tr>`;
    } else {
        body.innerHTML = rows.map(row => {
            const cells = cols.map(c => {
                const val = row[c];
                return `<td>${val === null || val === undefined ? 'NULL' : val}</td>`;
            }).join('');
            return `<tr>${cells}</tr>`;
        }).join('');
    }

    tableContainer.style.display = "block";
}
