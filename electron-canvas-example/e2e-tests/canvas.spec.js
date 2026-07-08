const { _electron: electron } = require('playwright');
const { test, expect } = require('@playwright/test');
const path = require('path');

test.describe('Electron Canvas Studio E2E Tests', () => {
  let electronApp;
  let window;

  test.beforeEach(async () => {
    // Launch Electron application using Playwright's built-in Electron capabilities
    electronApp = await electron.launch({
      args: [path.join(__dirname, '../main.js'), '--no-sandbox']
    });
    
    // Retrieve first loaded window
    window = await electronApp.firstWindow();
    await window.waitForLoadState('domcontentloaded');
  });

  test.afterEach(async () => {
    // Gracefully shut down the application
    if (electronApp) {
      await electronApp.close();
    }
  });

  test('should display the correct window title', async () => {
    const title = await window.title();
    expect(title).toBe('Electron Canvas Studio');
  });

  test('should load the sidebar with drawing tools', async () => {
    const sidebar = window.locator('.sidebar');
    await expect(sidebar).toBeVisible();

    // Verify critical tools are rendered
    const tools = ['pencil', 'line', 'rectangle', 'square', 'circle', 'eraser'];
    for (const tool of tools) {
      const btn = window.locator(`#tool-${tool}`);
      await expect(btn).toBeVisible();
    }

    // Verify Pencil is selected by default
    const pencilBtn = window.locator('#tool-pencil');
    await expect(pencilBtn).toHaveClass(/active/);
  });

  test('should change active tool on click', async () => {
    const lineBtn = window.locator('#tool-line');
    const statusTool = window.locator('#status-tool');

    await lineBtn.click();
    await expect(lineBtn).toHaveClass(/active/);
    await expect(statusTool).toHaveText('Tool: Line');

    const rectBtn = window.locator('#tool-rectangle');
    await rectBtn.click();
    await expect(rectBtn).toHaveClass(/active/);
    await expect(statusTool).toHaveText('Tool: Rectangle');
  });

  test('should toggle fill shape and enable/disable fill color inputs', async () => {
    const fillToggle = window.locator('.toggle-container');
    const fillColorGroup = window.locator('#fill-color-group');
    const fillColorInput = window.locator('#fill-color');

    // Fill options should start disabled
    await expect(fillColorGroup).toHaveClass(/disabled/);
    await expect(fillColorInput).toBeDisabled();

    // Toggle on fill
    await fillToggle.click();
    await expect(fillColorGroup).not.toHaveClass(/disabled/);
    await expect(fillColorInput).not.toBeDisabled();

    // Toggle off fill
    await fillToggle.click();
    await expect(fillColorGroup).toHaveClass(/disabled/);
    await expect(fillColorInput).toBeDisabled();
  });

  test('should perform drawing on canvas and register undo/clear', async () => {
    const canvas = window.locator('#drawing-canvas');
    await expect(canvas).toBeVisible();

    const box = await canvas.boundingBox();
    expect(box).not.toBeNull();

    // Draw standard lines on the canvas
    const startX = box.x + box.width * 0.3;
    const startY = box.y + box.height * 0.4;
    const endX = box.x + box.width * 0.5;
    const endY = box.y + box.height * 0.5;

    await window.mouse.move(startX, startY);
    await window.mouse.down();
    await window.mouse.move(endX, endY, { steps: 5 });
    await window.mouse.up();

    const undoBtn = window.locator('#action-undo');
    const clearBtn = window.locator('#action-clear');

    await expect(undoBtn).toBeVisible();
    await expect(clearBtn).toBeVisible();

    // Verify interactions don't break/crash the application
    await undoBtn.click();
    await clearBtn.click();
  });
});
