# Electron Canvas Studio

An elegant, interactive desktop drawing application built with **Electron**, **HTML5 Canvas**, and **pure vanilla JS/CSS**. It allows users to draw freehand curves and geometric shapes on a pixel-buffered workspace with full history support.

---

## Features

- **Drawing Canvas & Tools**:
  - **Pencil (P)**: Freehand drawing using pointer events for smooth stylus, pen, and mouse support.
  - **Line (L)**: Straight line rubber-banding.
  - **Rectangle (R)**: Dynamic rectangle drawing with outline and optional color fill.
  - **Square (S)**: Constrained square helper matching proportional height and width bounds.
  - **Circle (C)**: Radial circle generator drawing outward from the pointer down anchor.
  - **Eraser (E)**: Brush path using `destination-out` compositing to erase parts of the canvas cleanly.
- **Dynamic Sizing**: Uses a hidden offscreen context canvas to cache the buffer during window resizing. Your drawing remains perfectly intact if the window size changes.
- **Workspace Actions**:
  - **Undo / Redo (Ctrl+Z / Ctrl+Y)**: Dynamic historical state manager caching context copies.
  - **Clear Canvas**: Resets the workspace.
  - **Save PNG (Ctrl+S)**: Downloads the drawing buffer directly as a transparent PNG file.
- **Color & Stroke Properties**:
  - Main brush stroke color selection.
  - Brush thickness/width slider (1px - 50px) with live Badge view.
  - Fill Shape checkbox and corresponding Fill color picker (toggles active state dynamically).
- **Aesthetic Dark Theme UI**: Glassmorphic panels with blur overlays, violet accent tones, cursor cues corresponding to active tools, coordinate tracks, and dotted grid workspace style background.

---

## Keyboard Shortcuts

- `P` : Select Pencil (Freehand)
- `L` : Select Line
- `R` : Select Rectangle
- `S` : Select Square
- `C` : Select Circle
- `E` : Select Eraser
- `Ctrl + Z` : Undo last action
- `Ctrl + Y` : Redo undone action
- `Ctrl + S` : Save canvas to PNG

---

## Getting Started

### Prerequisites

Make sure you have [Node.js](https://nodejs.org) installed on your system.

### Installation

Navigate to the project folder and install the dependencies:

```bash
cd electron-canvas-example
npm install
```

### Running the Application

To boot up the Electron Canvas Studio desktop window:

```bash
npm start
```

---

## Testing Suite

The project includes an automated test framework combining **Jasmine** (for unit/algorithmic verification) and **Playwright** (for E2E desktop integration testing).

### Running all tests

To run both unit and integration tests sequentially:

```bash
npm test
```

### Unit Tests (Jasmine + JSDOM)

Runs validation against internal drawing algorithms, checkbox interactions, and history stacks by mocking canvas element contexts. No native GUI elements or display servers are needed:

```bash
npm run test:unit
```

### E2E Integration Tests (Playwright)

Launches the native Electron process and validates panel render states, clicking triggers, property updates, drag-and-draw coordinate actions, and canvas undo registers:

```bash
npm run test:e2e
```
