const fs = require('fs');
const path = require('path');
const { JSDOM } = require('jsdom');

describe('Renderer Drawing Logic Unit Tests', () => {
  let dom;
  let window;
  let document;

  beforeEach(() => {
    // Read the HTML template
    const htmlPath = path.join(__dirname, '../index.html');
    const html = fs.readFileSync(htmlPath, 'utf8');

    // Create the virtual DOM
    dom = new JSDOM(html, {
      runScripts: 'dangerously',
      resources: 'usable'
    });
    window = dom.window;
    document = window.document;

    // Mock HTMLCanvasElement context to avoid native platform binary canvas dependencies
    const mockCtx = jasmine.createSpyObj('CanvasRenderingContext2D', [
      'beginPath', 'moveTo', 'lineTo', 'stroke', 'rect', 'arc', 'fill', 'clearRect', 'drawImage'
    ]);
    
    // Default context state properties
    mockCtx.strokeStyle = '#000000';
    mockCtx.fillStyle = '#000000';
    mockCtx.lineWidth = 1;
    mockCtx.lineCap = 'round';
    mockCtx.lineJoin = 'round';
    mockCtx.globalCompositeOperation = 'source-over';

    window.HTMLCanvasElement.prototype.getContext = function(type) {
      if (type === '2d') {
        return mockCtx;
      }
      return null;
    };

    // Load and execute renderer.js in JSDOM context
    const rendererCode = fs.readFileSync(path.join(__dirname, '../renderer.js'), 'utf8');
    const scriptEl = document.createElement('script');
    scriptEl.textContent = rendererCode;
    document.body.appendChild(scriptEl);
  });

  afterEach(() => {
    if (window) {
      window.close();
    }
  });

  it('should initialize with Pencil tool selected', () => {
    expect(window.activeTool).toBe('pencil');
    expect(document.getElementById('tool-pencil').classList.contains('active')).toBe(true);
  });

  it('should change active tool when selectTool is called', () => {
    window.selectTool('line');
    expect(window.activeTool).toBe('line');
    expect(document.getElementById('tool-line').classList.contains('active')).toBe(true);
    expect(document.getElementById('tool-pencil').classList.contains('active')).toBe(false);
  });

  it('should correctly configure canvas context properties', () => {
    window.strokeColor = '#ff0000';
    window.strokeWidth = 10;
    
    window.configureContext();
    const ctx = document.getElementById('drawing-canvas').getContext('2d');
    
    expect(ctx.strokeStyle).toBe('#ff0000');
    expect(ctx.lineWidth).toBe(10);
  });

  it('should draw shapes correctly on canvas context', () => {
    const canvas = document.getElementById('drawing-canvas');
    const ctx = canvas.getContext('2d');

    // Draw line
    window.drawShape('line', { x: 10, y: 10 }, { x: 50, y: 50 });
    expect(ctx.beginPath).toHaveBeenCalled();
    expect(ctx.moveTo).toHaveBeenCalledWith(10, 10);
    expect(ctx.lineTo).toHaveBeenCalledWith(50, 50);
    expect(ctx.stroke).toHaveBeenCalled();

    // Reset calls
    ctx.beginPath.calls.reset();
    ctx.rect.calls.reset();
    
    // Draw rectangle (no fill)
    window.fillEnabled = false;
    window.drawShape('rectangle', { x: 20, y: 20 }, { x: 60, y: 80 });
    expect(ctx.beginPath).toHaveBeenCalled();
    expect(ctx.rect).toHaveBeenCalledWith(20, 20, 40, 60);
    expect(ctx.fill).not.toHaveBeenCalled();
    expect(ctx.stroke).toHaveBeenCalled();
  });

  it('should draw a constrained square when square tool is selected', () => {
    const canvas = document.getElementById('drawing-canvas');
    const ctx = canvas.getContext('2d');

    // Draw square with drag width=40, height=80 (should constrain to 40x40 side)
    window.drawShape('square', { x: 10, y: 10 }, { x: 50, y: 90 });
    expect(ctx.rect).toHaveBeenCalledWith(10, 10, 40, 40);
  });

  it('should clear canvas and update history on triggerClear', () => {
    const canvas = document.getElementById('drawing-canvas');
    const ctx = canvas.getContext('2d');

    window.triggerClear();
    expect(ctx.clearRect).toHaveBeenCalled();
    expect(window.undoStack.length).toBe(1);
  });
});
