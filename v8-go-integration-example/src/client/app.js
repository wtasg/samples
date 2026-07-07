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
const templateBtns = document.querySelectorAll('.tmpl-btn');

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

let activeTemplate = 'math';

// Initialize Editor
function initEditor() {
    codeEditor.value = templates[activeTemplate];
    updateLineNumbers();
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

codeEditor.addEventListener('input', updateLineNumbers);

// Template Button Listeners
templateBtns.forEach(btn => {
    btn.addEventListener('click', (e) => {
        templateBtns.forEach(b => b.classList.remove('active'));
        btn.classList.add('active');
        activeTemplate = btn.dataset.template;
        codeEditor.value = templates[activeTemplate];
        updateLineNumbers();
        resetVisualFlow();
    });
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

// Event Listeners
runBtn.addEventListener('click', runScript);

// Ctrl + Enter shortcut
document.addEventListener('keydown', (e) => {
    if (e.ctrlKey && e.key === 'Enter') {
        e.preventDefault();
        runScript();
    }
});

// Initialize on page load
initEditor();
