// DOM Elements
const canvas = document.getElementById('drawing-canvas');
const ctx = canvas.getContext('2d');

const tools = {
  pencil: document.getElementById('tool-pencil'),
  line: document.getElementById('tool-line'),
  rectangle: document.getElementById('tool-rectangle'),
  square: document.getElementById('tool-square'),
  circle: document.getElementById('tool-circle'),
  eraser: document.getElementById('tool-eraser')
};

const actionUndo = document.getElementById('action-undo');
const actionRedo = document.getElementById('action-redo');
const actionClear = document.getElementById('action-clear');
const actionSave = document.getElementById('action-save');

const strokeColorInput = document.getElementById('stroke-color');
const strokeColorPreview = document.getElementById('stroke-color-preview');
const brushSizeInput = document.getElementById('brush-size');
const brushSizeBadge = document.getElementById('brush-size-badge');

const fillShapeCheckbox = document.getElementById('fill-shape');
const fillColorInput = document.getElementById('fill-color');
const fillColorPreview = document.getElementById('fill-color-preview');
const fillColorGroup = document.getElementById('fill-color-group');

const statusTool = document.getElementById('status-tool');
const statusCoords = document.getElementById('status-coords');
const statusSize = document.getElementById('status-size');

// Drawing state
let activeTool = 'pencil';
let isDrawing = false;
let startPoint = { x: 0, y: 0 };
let currentPoint = { x: 0, y: 0 };

let strokeColor = strokeColorInput.value;
let strokeWidth = parseInt(brushSizeInput.value);
let fillEnabled = fillShapeCheckbox.checked;
let fillColor = fillColorInput.value;

// Canvas state snapshots for rubber-banding and undo/redo
let startStateCanvas = null; // Canvas snap taken at pointerdown
let undoStack = [];
let redoStack = [];
const MAX_HISTORY = 40;

// Initialize Canvas Sizing
function initCanvas() {
  resizeCanvas();
  // Adjust sizing on window resize
  window.addEventListener('resize', resizeCanvas);
}

// Resizes canvas safely without losing the drawing context
function resizeCanvas() {
  const container = document.getElementById('canvas-container');
  const rect = container.getBoundingClientRect();

  // Create offscreen copy of the current drawing
  const tempCanvas = document.createElement('canvas');
  tempCanvas.width = canvas.width;
  tempCanvas.height = canvas.height;
  const tempCtx = tempCanvas.getContext('2d');
  tempCtx.drawImage(canvas, 0, 0);

  // Resize the main canvas
  canvas.width = rect.width;
  canvas.height = rect.height;

  // Restore the drawing
  configureContext();
  ctx.drawImage(tempCanvas, 0, 0);

  // Update Status bar
  statusSize.textContent = `Canvas: ${canvas.width} x ${canvas.height}`;
}

// Configures standard canvas drawing properties
function configureContext() {
  ctx.strokeStyle = strokeColor;
  ctx.lineWidth = strokeWidth;
  ctx.lineCap = 'round';
  ctx.lineJoin = 'round';
  ctx.globalCompositeOperation = 'source-over';
}

// Selects active drawing tool
function selectTool(toolName) {
  if (!tools[toolName]) return;

  // Update active style classes
  Object.values(tools).forEach(btn => btn.classList.remove('active'));
  tools[toolName].classList.add('active');

  activeTool = toolName;
  statusTool.textContent = `Tool: ${toolName.charAt(0).toUpperCase() + toolName.slice(1)}`;

  // Set visual cursor
  if (toolName === 'eraser') {
    canvas.style.cursor = 'cell';
  } else {
    canvas.style.cursor = 'crosshair';
  }
}

// Undo/Redo Engine
function saveHistoryState() {
  // Clear the redo stack since a new action took place
  redoStack = [];

  const stateCanvas = document.createElement('canvas');
  stateCanvas.width = canvas.width;
  stateCanvas.height = canvas.height;
  const stateCtx = stateCanvas.getContext('2d');
  stateCtx.drawImage(canvas, 0, 0);

  undoStack.push(stateCanvas);
  if (undoStack.length > MAX_HISTORY) {
    undoStack.shift();
  }
}

function triggerUndo() {
  if (undoStack.length === 0) return;

  // Capture current state to push onto redo stack
  const currentCanvas = document.createElement('canvas');
  currentCanvas.width = canvas.width;
  currentCanvas.height = canvas.height;
  const currentCtx = currentCanvas.getContext('2d');
  currentCtx.drawImage(canvas, 0, 0);
  redoStack.push(currentCanvas);

  // Pop and draw the previous state
  const prevStateCanvas = undoStack.pop();
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(prevStateCanvas, 0, 0);
}

function triggerRedo() {
  if (redoStack.length === 0) return;

  // Capture current state to push onto undo stack
  const currentCanvas = document.createElement('canvas');
  currentCanvas.width = canvas.width;
  currentCanvas.height = canvas.height;
  const currentCtx = currentCanvas.getContext('2d');
  currentCtx.drawImage(canvas, 0, 0);
  undoStack.push(currentCanvas);

  // Pop and draw the next state
  const nextStateCanvas = redoStack.pop();
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(nextStateCanvas, 0, 0);
}

function triggerClear() {
  saveHistoryState();
  ctx.clearRect(0, 0, canvas.width, canvas.height);
}

function triggerSave() {
  const dataURL = canvas.toDataURL('image/png');
  const link = document.createElement('a');
  link.download = `canvas-studio-${Date.now()}.png`;
  link.href = dataURL;
  link.click();
}

// Shape rendering helper
function drawShape(tool, start, current) {
  configureContext();

  switch (tool) {
    case 'line':
      ctx.beginPath();
      ctx.moveTo(start.x, start.y);
      ctx.lineTo(current.x, current.y);
      ctx.stroke();
      break;

    case 'rectangle': {
      const w = current.x - start.x;
      const h = current.y - start.y;
      ctx.beginPath();
      ctx.rect(start.x, start.y, w, h);
      if (fillEnabled) {
        ctx.fillStyle = fillColor;
        ctx.fill();
      }
      ctx.stroke();
      break;
    }

    case 'square': {
      let w = current.x - start.x;
      let h = current.y - start.y;
      const side = Math.min(Math.abs(w), Math.abs(h));
      w = Math.sign(w) * side;
      h = Math.sign(h) * side;
      ctx.beginPath();
      ctx.rect(start.x, start.y, w, h);
      if (fillEnabled) {
        ctx.fillStyle = fillColor;
        ctx.fill();
      }
      ctx.stroke();
      break;
    }

    case 'circle': {
      const dx = current.x - start.x;
      const dy = current.y - start.y;
      const radius = Math.sqrt(dx * dx + dy * dy);
      ctx.beginPath();
      ctx.arc(start.x, start.y, radius, 0, 2 * Math.PI);
      if (fillEnabled) {
        ctx.fillStyle = fillColor;
        ctx.fill();
      }
      ctx.stroke();
      break;
    }
  }
}

// Pointer Event Helpers
function getPointerPos(e) {
  const rect = canvas.getBoundingClientRect();
  return {
    x: e.clientX - rect.left,
    y: e.clientY - rect.top
  };
}

// Core drawing pointer listeners
canvas.addEventListener('pointerdown', (e) => {
  if (e.button !== 0) return; // Only trigger drawing on main/left mouse click

  isDrawing = true;
  startPoint = getPointerPos(e);
  currentPoint = { ...startPoint };

  // Save history state for undo operations
  saveHistoryState();

  // Create snapshot before drawing stroke (for shape rubber-banding)
  startStateCanvas = document.createElement('canvas');
  startStateCanvas.width = canvas.width;
  startStateCanvas.height = canvas.height;
  const startStateCtx = startStateCanvas.getContext('2d');
  startStateCtx.drawImage(canvas, 0, 0);

  configureContext();

  if (activeTool === 'pencil' || activeTool === 'eraser') {
    if (activeTool === 'eraser') {
      ctx.globalCompositeOperation = 'destination-out';
    } else {
      ctx.globalCompositeOperation = 'source-over';
    }
    ctx.beginPath();
    ctx.moveTo(startPoint.x, startPoint.y);
    // Draw initial dot in case of simple tap/click
    ctx.lineTo(startPoint.x, startPoint.y);
    ctx.stroke();
  }

  canvas.setPointerCapture(e.pointerId);
});

canvas.addEventListener('pointermove', (e) => {
  const pos = getPointerPos(e);
  statusCoords.textContent = `X: ${Math.round(pos.x)}px, Y: ${Math.round(pos.y)}px`;

  if (!isDrawing) return;

  currentPoint = pos;

  if (activeTool === 'pencil' || activeTool === 'eraser') {
    if (activeTool === 'eraser') {
      ctx.globalCompositeOperation = 'destination-out';
    } else {
      ctx.globalCompositeOperation = 'source-over';
    }
    ctx.lineTo(currentPoint.x, currentPoint.y);
    ctx.stroke();
  } else {
    // Clear canvas back to state before stroke started, and redraw shape
    ctx.globalCompositeOperation = 'source-over';
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(startStateCanvas, 0, 0);
    drawShape(activeTool, startPoint, currentPoint);
  }
});

canvas.addEventListener('pointerup', (e) => {
  if (!isDrawing) return;

  isDrawing = false;
  canvas.releasePointerCapture(e.pointerId);
  startStateCanvas = null;
});

canvas.addEventListener('pointercancel', (e) => {
  if (!isDrawing) return;

  isDrawing = false;
  canvas.releasePointerCapture(e.pointerId);
  startStateCanvas = null;

  // Restore state back to before canceled stroke began
  if (undoStack.length > 0) {
    const prevState = undoStack.pop();
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(prevState, 0, 0);
  }
});

// Event Listeners for Tool buttons
Object.entries(tools).forEach(([toolName, element]) => {
  element.addEventListener('click', () => selectTool(toolName));
});

// Event Listeners for Actions
actionUndo.addEventListener('click', triggerUndo);
actionRedo.addEventListener('click', triggerRedo);
actionClear.addEventListener('click', triggerClear);
actionSave.addEventListener('click', triggerSave);

// Property Controls Change Listeners
strokeColorInput.addEventListener('input', (e) => {
  strokeColor = e.target.value;
  strokeColorPreview.style.backgroundColor = strokeColor;
});

brushSizeInput.addEventListener('input', (e) => {
  strokeWidth = parseInt(e.target.value);
  brushSizeBadge.textContent = `${strokeWidth}px`;
});

fillShapeCheckbox.addEventListener('change', (e) => {
  fillEnabled = e.target.checked;
  if (fillEnabled) {
    fillColorGroup.classList.remove('disabled');
    fillColorInput.removeAttribute('disabled');
    fillColorPreview.classList.remove('disabled');
  } else {
    fillColorGroup.classList.add('disabled');
    fillColorInput.setAttribute('disabled', 'true');
    fillColorPreview.classList.add('disabled');
  }
});

fillColorInput.addEventListener('input', (e) => {
  fillColor = e.target.value;
  fillColorPreview.style.backgroundColor = fillColor;
});

// Keyboard Shortcuts Integration
window.addEventListener('keydown', (e) => {
  // Prevent shortcuts while focusing on color selectors or slider controls
  if (e.target.tagName === 'INPUT') return;

  const isCmdOrCtrl = e.ctrlKey || e.metaKey;

  if (isCmdOrCtrl && e.key.toLowerCase() === 'z') {
    e.preventDefault();
    triggerUndo();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 'y') {
    e.preventDefault();
    triggerRedo();
  } else if (isCmdOrCtrl && e.key.toLowerCase() === 's') {
    e.preventDefault();
    triggerSave();
  } else {
    switch (e.key.toLowerCase()) {
      case 'p':
        selectTool('pencil');
        break;
      case 'l':
        selectTool('line');
        break;
      case 'r':
        selectTool('rectangle');
        break;
      case 's':
        selectTool('square');
        break;
      case 'c':
        selectTool('circle');
        break;
      case 'e':
        selectTool('eraser');
        break;
    }
  }
});

// Initialize on page load
window.addEventListener('DOMContentLoaded', () => {
  initCanvas();
  selectTool('pencil');
});

// Expose internal state and helpers on the window object for automated unit testing
if (typeof window !== 'undefined') {
  Object.defineProperty(window, 'activeTool', {
    get: () => activeTool,
    set: (val) => { activeTool = val; }
  });
  Object.defineProperty(window, 'strokeColor', {
    get: () => strokeColor,
    set: (val) => { strokeColor = val; }
  });
  Object.defineProperty(window, 'strokeWidth', {
    get: () => strokeWidth,
    set: (val) => { strokeWidth = val; }
  });
  Object.defineProperty(window, 'fillEnabled', {
    get: () => fillEnabled,
    set: (val) => { fillEnabled = val; }
  });
  Object.defineProperty(window, 'fillColor', {
    get: () => fillColor,
    set: (val) => { fillColor = val; }
  });

  window.selectTool = selectTool;
  window.undoStack = undoStack;
  window.redoStack = redoStack;
  window.configureContext = configureContext;
  window.drawShape = drawShape;
  window.triggerClear = triggerClear;
}

