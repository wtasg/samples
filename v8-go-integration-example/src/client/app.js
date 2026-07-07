// Code Templates
const templates = {
    math: `// Basic calculation executed inside the sandboxed V8 context
const radius = 12;
const area = Math.PI * Math.pow(radius, 2);

console.log("Starting calculation in V8...");
console.log("Radius: " + radius);
console.log("Calculated Area: " + area.toFixed(4));

// The final evaluated statement is returned to Go
\`Circle Area is: \${area.toFixed(4)} sq units\`;`,

    gocompute: `// Demonstration of JavaScript calling Go back
// The 'goCompute' function is injected into the V8 global environment by Go

console.log("Starting JS script in V8...");
console.log("V8 is now delegating math computation (15 * 30) to Go...");

// Calling the Go-side callback
const result = goCompute(15, 30);

console.log("Go computed multiplication result successfully!");
\`Result computed in Go: \${result}\`;`,

    gofetch: `// Demonstration of V8 calling Go to fetch data
// 'goFetch' triggers Go-side functions to query resources and return data

console.log("JS script started in V8 isolate...");
console.log("Initiating backend API call via Go's HTTP client...");

// Call the Go function
const rawResponse = goFetch("https://api.sample.org/v1/data");

console.log("Raw JSON response string received from Go!");
console.log("JSON String: " + rawResponse);

// Parse and extract values inside Javascript
const payload = JSON.parse(rawResponse);
console.log("Successfully parsed JSON payload inside V8");

\`Fetched URL: \${payload.url}\\nStatus: \${payload.status}\\nRetrieved Go-themed items: \${payload.data.items.join(', ')}\`;`,

    error: `// Demonstration of runtime error propagation
// V8-Go integration captures JS errors, mapping stack traces back to Go

console.log("Starting run...");
console.log("Calling undefined function to trigger error...");

function executeSubtask() {
    console.log("Inside executeSubtask() - throwing error");
    // This function does not exist
    triggerErrorInV8Engine();
}

executeSubtask();`
};

// Elements
const codeEditor = document.getElementById('code-editor');
const lineNumbers = document.getElementById('line-numbers');
const runBtn = document.getElementById('run-btn');
const btnSpinner = document.getElementById('btn-spinner');
const statusBadge = document.getElementById('status-badge');
const executionTime = document.getElementById('execution-time');
const consoleLogs = document.getElementById('console-logs');
const returnedValue = document.getElementById('returned-value');
const clearConsoleBtn = document.getElementById('clear-console-btn');
const editorTabsContainer = document.getElementById('editor-tabs');
const newTabBtn = document.getElementById('new-tab-btn');

// Flow Diagram Elements
const flowNodes = {
    web: document.getElementById('node-web'),
    server: document.getElementById('node-server'),
    lib: document.getElementById('node-lib'),
    v8: document.getElementById('node-v8'),
    back: document.getElementById('node-back'),
};
const flowArrows = {
    webServer: document.getElementById('arrow-web-server'),
    serverLib: document.getElementById('arrow-server-lib'),
    libV8: document.getElementById('arrow-lib-v8'),
    v8Callback: document.getElementById('arrow-v8-callback'),
    v8Lib: document.getElementById('arrow-v8-lib'),
};

// Tabbed Workspace State
let activeTab = 'math';
let nextScratchId = 1;

const tabContents = {
    math: templates.math,
    gocompute: templates.gocompute,
    gofetch: templates.gofetch,
    error: templates.error
};

const tabMeta = {
    math: { title: 'Basic Math', closeable: false },
    gocompute: { title: 'Go Callback (Math)', closeable: false },
    gofetch: { title: 'Go Callback (Fetch)', closeable: false },
    error: { title: 'JS Error Handler', closeable: false }
};

// Initialize Tabs UI
function renderTabs() {
    if (!editorTabsContainer) return;
    editorTabsContainer.innerHTML = '';
    
    Object.keys(tabMeta).forEach(tabId => {
        const meta = tabMeta[tabId];
        const btn = document.createElement('button');
        btn.className = `tmpl-btn ${activeTab === tabId ? 'active' : ''}`;
        btn.dataset.tab = tabId;
        
        const titleSpan = document.createElement('span');
        titleSpan.textContent = meta.title;
        btn.appendChild(titleSpan);
        
        if (meta.closeable) {
            const closeSpan = document.createElement('span');
            closeSpan.className = 'tab-close';
            closeSpan.innerHTML = '&times;';
            closeSpan.title = 'Close editor tab';
            closeSpan.addEventListener('click', (e) => {
                e.stopPropagation();
                closeTab(tabId);
            });
            btn.appendChild(closeSpan);
        }
        
        btn.addEventListener('click', () => {
            switchTab(tabId);
        });
        
        editorTabsContainer.appendChild(btn);
    });
}

// Switch Editor Tab
function switchTab(tabId) {
    if (activeTab === tabId) return;
    // Save current content
    tabContents[activeTab] = codeEditor.value;
    
    activeTab = tabId;
    codeEditor.value = tabContents[tabId];
    renderTabs();
    updateLineNumbers();
    resetVisualFlow();
}

// Close Editor Tab
function closeTab(tabId) {
    if (!tabMeta[tabId] || !tabMeta[tabId].closeable) return;
    
    delete tabContents[tabId];
    delete tabMeta[tabId];
    
    if (activeTab === tabId) {
        const remainingKeys = Object.keys(tabMeta);
        activeTab = remainingKeys[remainingKeys.length - 1];
    }
    
    codeEditor.value = tabContents[activeTab];
    renderTabs();
    updateLineNumbers();
    resetVisualFlow();
}

// Create New Blank Editor Tab
function createNewTab() {
    // Save current
    tabContents[activeTab] = codeEditor.value;
    
    const tabId = `scratch-${nextScratchId}`;
    const tabTitle = `Untitled ${nextScratchId}`;
    nextScratchId++;
    
    tabContents[tabId] = `// ${tabTitle} - Code from scratch inside V8 isolate\n\nconsole.log("Running custom code...");\n\n"Hello World!";`;
    tabMeta[tabId] = { title: tabTitle, closeable: true };
    
    activeTab = tabId;
    codeEditor.value = tabContents[tabId];
    renderTabs();
    updateLineNumbers();
    resetVisualFlow();
    
    // Focus the editor
    codeEditor.focus();
}

// Update Line Numbers
function updateLineNumbers() {
    const lines = codeEditor.value.split('\n');
    const numbers = lines.map((_, index) => index + 1).join('<br>');
    lineNumbers.innerHTML = numbers;
}

// Sync Scrolling between editor and line numbers
codeEditor.addEventListener('scroll', () => {
    lineNumbers.scrollTop = codeEditor.scrollTop;
});

// Update content cache as the user types
codeEditor.addEventListener('input', () => {
    tabContents[activeTab] = codeEditor.value;
    updateLineNumbers();
});

// Clear console
clearConsoleBtn.addEventListener('click', () => {
    consoleLogs.innerHTML = '<div class="console-placeholder">Console cleared. Ready for execution.</div>';
});

// Reset Visual Flow Diagram
function resetVisualFlow() {
    Object.values(flowNodes).forEach(node => node.classList.remove('active'));
    Object.values(flowArrows).forEach(arrow => arrow.classList.remove('active'));
    flowArrows.v8Callback.classList.add('hidden');
}

// Visual Flow Pipeline Animation Sequence
function animatePipeline(isCallbackMode, callbackDelay = 500) {
    resetVisualFlow();
    
    // Step 1: Web Client starts
    flowNodes.web.classList.add('active');
    
    // Step 2: Request travels to Go Server
    setTimeout(() => {
        flowArrows.webServer.classList.add('active');
        flowNodes.server.classList.add('active');
    }, 150);

    // Step 3: Server delegates to Decoupled Interface
    setTimeout(() => {
        flowArrows.serverLib.classList.add('active');
        flowNodes.lib.classList.add('active');
    }, 300);

    // Step 4: Interface starts V8 engine
    setTimeout(() => {
        flowArrows.libV8.classList.add('active');
        flowNodes.v8.classList.add('active');
    }, 450);

    // Step 5: (Optional) V8 runs script and callbacks to Go
    if (isCallbackMode) {
        setTimeout(() => {
            flowArrows.v8Callback.classList.remove('hidden');
            flowArrows.v8Callback.classList.add('active');
        }, 600);
    }
}

// Finish Visual Flow
function finishPipeline(success) {
    // Light up return path
    setTimeout(() => {
        flowArrows.v8Lib.classList.add('active');
        flowNodes.back.classList.add('active');
    }, 100);
}

// Run Script
async function runScript() {
    const script = codeEditor.value.trim();
    if (!script) return;

    // Check if script uses Go callbacks
    const isCallbackMode = script.includes('goCompute') || script.includes('goFetch');

    // UI state: Running
    runBtn.disabled = true;
    btnSpinner.style.display = 'block';
    statusBadge.className = 'badge running';
    statusBadge.textContent = 'Running';
    
    animatePipeline(isCallbackMode);

    try {
        const response = await fetch('/api/run', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ script })
        });

        if (!response.ok) {
            throw new Error(`Server returned HTTP ${response.status}`);
        }

        const data = await response.json();

        // UI state: Done
        finishPipeline(data.success);
        
        executionTime.textContent = `${data.duration_ms} ms`;

        // Render console output
        renderConsole(data.logs);

        // Render final value
        returnedValue.className = 'result-content';
        if (data.success) {
            statusBadge.className = 'badge success';
            statusBadge.textContent = 'Success';
            returnedValue.classList.add('success');
            returnedValue.textContent = data.result !== "" ? data.result : "undefined";
        } else {
            statusBadge.className = 'badge error';
            statusBadge.textContent = 'JS Error';
            returnedValue.classList.add('error');
            returnedValue.textContent = data.error;
        }

    } catch (err) {
        resetVisualFlow();
        statusBadge.className = 'badge error';
        statusBadge.textContent = 'Server Error';
        executionTime.textContent = 'Error';
        returnedValue.className = 'result-content error';
        returnedValue.textContent = `Server Connection Failure: ${err.message}`;
        renderConsole([`[SYSTEM ERROR] Failed to connect to backend: ${err.message}`]);
    } finally {
        runBtn.disabled = false;
        btnSpinner.style.display = 'none';
    }
}

// Render Console Logs helper
function renderConsole(logs) {
    if (!logs || logs.length === 0) {
        consoleLogs.innerHTML = '<div class="console-placeholder">Execution completed with no console logs.</div>';
        return;
    }

    consoleLogs.innerHTML = '';
    logs.forEach(log => {
        const line = document.createElement('div');
        line.className = 'console-line';
        
        if (log.startsWith('[Go Callback]')) {
            line.classList.add('callback');
        } else if (log.startsWith('[SYSTEM ERROR]')) {
            line.classList.add('error');
        } else {
            line.classList.add('js-log');
            // Prefix to make clear it is a JS console log
            log = `❯ ${log}`;
        }
        
        line.textContent = log;
        consoleLogs.appendChild(line);
    });
    
    // Auto Scroll to bottom
    consoleLogs.scrollTop = consoleLogs.scrollHeight;
}

// Drag Resizing Pane logic
const workspace = document.querySelector('.workspace');
const dragbar = document.getElementById('dragbar');
let isDragging = false;

function initResize() {
    if (!dragbar || !workspace) return;

    dragbar.addEventListener('mousedown', (e) => {
        isDragging = true;
        dragbar.classList.add('dragging');
        document.body.style.cursor = 'col-resize';
        document.body.style.userSelect = 'none';
    });

    document.addEventListener('mousemove', (e) => {
        if (!isDragging) return;
        
        const containerRect = workspace.getBoundingClientRect();
        const containerWidth = containerRect.width;
        
        let offset = e.clientX - containerRect.left;
        let percentage = (offset / containerWidth) * 100;
        
        // Impose constraints (e.g., 20% to 80%)
        if (percentage < 20) percentage = 20;
        if (percentage > 80) percentage = 80;
        
        workspace.style.gridTemplateColumns = `${percentage}% 10px ${100 - percentage}%`;
    });

    document.addEventListener('mouseup', () => {
        if (isDragging) {
            isDragging = false;
            dragbar.classList.remove('dragging');
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        }
    });

    // Touch Support for Mobile
    dragbar.addEventListener('touchstart', (e) => {
        isDragging = true;
        dragbar.classList.add('dragging');
        document.body.style.userSelect = 'none';
    });

    document.addEventListener('touchmove', (e) => {
        if (!isDragging) return;
        
        const clientX = e.touches[0].clientX;
        const containerRect = workspace.getBoundingClientRect();
        const containerWidth = containerRect.width;
        
        let offset = clientX - containerRect.left;
        let percentage = (offset / containerWidth) * 100;
        
        if (percentage < 20) percentage = 20;
        if (percentage > 80) percentage = 80;
        
        workspace.style.gridTemplateColumns = `${percentage}% 10px ${100 - percentage}%`;
    });

    document.addEventListener('touchend', () => {
        if (isDragging) {
            isDragging = false;
            dragbar.classList.remove('dragging');
            document.body.style.userSelect = '';
        }
    });
}

// Event Listeners
runBtn.addEventListener('click', runScript);

if (newTabBtn) {
    newTabBtn.addEventListener('click', createNewTab);
}

// Ctrl + Enter shortcut
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 'Enter') {
        e.preventDefault();
        runScript();
    }
});

// Initialize on page load
function init() {
    codeEditor.value = tabContents[activeTab];
    renderTabs();
    updateLineNumbers();
    initResize();
}

init();
