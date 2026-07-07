import { test, expect } from '@playwright/test';

test.describe('V8 + Go Integration Playground E2E Tests', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
  });

  test('should load playground title and columns', async ({ page }) => {
    await expect(page).toHaveTitle('V8 + Go Integration Playground');
    await expect(page.locator('h1')).toHaveText('V8 Engine + Go Integration');
    
    // Check columns
    await expect(page.locator('.editor-card')).toBeVisible();
    await expect(page.locator('.results-card')).toBeVisible();
    await expect(page.locator('#dragbar')).toBeVisible();
  });

  test('should execute basic math template successfully', async ({ page }) => {
    // Wait for badge idle state
    await expect(page.locator('#status-badge')).toHaveText('Idle');
    
    // Execute
    await page.click('#run-btn');
    
    // Wait for execution success
    await expect(page.locator('#status-badge')).toHaveText('Success');
    
    // Verify console logs
    const consoleContent = page.locator('#console-logs');
    await expect(consoleContent).toContainText('Starting calculation in V8...');
    await expect(consoleContent).toContainText('Radius: 12');
    
    // Verify final returned value
    const returnedValue = page.locator('#returned-value');
    await expect(returnedValue).toContainText('Circle Area is: 452.3893 sq units');
  });

  test('should allow custom tabs creation and execution from scratch', async ({ page }) => {
    // Click "+ New"
    await page.click('#new-tab-btn');
    
    // A tab called "Untitled 1" should be active
    const activeTab = page.locator('.tmpl-btn.active');
    await expect(activeTab).toContainText('Untitled 1');
    
    // Modify text in editor
    const editor = page.locator('#code-editor');
    await editor.fill('console.log("Playwright custom script!");\n\n"Custom Value 789";');
    
    // Click Execute
    await page.click('#run-btn');
    
    // Verify Execution
    await expect(page.locator('#status-badge')).toHaveText('Success');
    await expect(page.locator('#console-logs')).toContainText('Playwright custom script!');
    await expect(page.locator('#returned-value')).toHaveText('Custom Value 789');
    
    // Close the tab
    await page.click('.tmpl-btn.active .tab-close');
    
    // Active tab should return to last template
    await expect(page.locator('.tmpl-btn.active')).toHaveText('JS Error Handler');
  });

  test('should preserve code state when switching tabs', async ({ page }) => {
    // Create tab 1
    await page.click('#new-tab-btn');
    const editor = page.locator('#code-editor');
    await editor.fill('// Tab 1 unique script content');
    
    // Create tab 2
    await page.click('#new-tab-btn');
    await editor.fill('// Tab 2 unique script content');
    
    // Switch back to Tab 1
    await page.click('button.tmpl-btn:has-text("Untitled 1")');
    await expect(editor).toHaveValue('// Tab 1 unique script content');
    
    // Switch to Tab 2
    await page.click('button.tmpl-btn:has-text("Untitled 2")');
    await expect(editor).toHaveValue('// Tab 2 unique script content');
  });

  test('should support panel resizing by dragging the resizer bar', async ({ page }) => {
    const editorCard = page.locator('.editor-card');
    
    // Get initial width
    const initialBox = await editorCard.boundingBox();
    expect(initialBox).not.toBeNull();
    const initialWidth = initialBox!.width;
    
    // Drag resizer
    const dragbar = page.locator('#dragbar');
    const dragbarBox = await dragbar.boundingBox();
    expect(dragbarBox).not.toBeNull();
    
    const startX = dragbarBox!.x + dragbarBox!.width / 2;
    const startY = dragbarBox!.y + dragbarBox!.height / 2;
    
    // Drag left by 150px
    await page.mouse.move(startX, startY);
    await page.mouse.down();
    await page.mouse.move(startX - 150, startY, { steps: 10 });
    await page.mouse.up();
    
    // Get new width
    const newBox = await editorCard.boundingBox();
    expect(newBox).not.toBeNull();
    const newWidth = newBox!.width;
    
    // The editor card should have resized
    expect(newWidth).toBeLessThan(initialWidth);
  });
});
