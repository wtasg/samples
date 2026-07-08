# Graphviz Flow - Interactive SVG Diagram Studio

A Python FastAPI web application for dynamically designing, rendering, and manipulating Graphviz SVG diagrams. The project features an interactive split-screen workspace where users can write DOT code and interactively edit the diagram using drag-and-drop actions.

## System Architecture

```text
User <---> Web Server (FastAPI) <---> Graphviz (CLI) <---> SVG Output
```

1. **Code Edits**: Typing in the DOT code editor sends code to the backend. The backend compiles the DOT source using `dot -Tsvg` and `dot -Tjson`, returning the SVG layout and coordinate details to the frontend.
2. **Visual Interaction**: The frontend parses the SVG DOM, enabling users to:
   - Hover over nodes to reveal "connector handles". Dragging from a handle onto another node creates an edge connection.
   - Drag nodes to relocate them (in coordinate-based layouts like `neato` or `fdp`), which automatically updates their `pos` attributes in the DOT editor.
   - Drag shapes from the sidebar palette onto the canvas to insert new nodes.
   - Select nodes or edges to modify labels, shapes, and colors via the Properties Inspector.
3. All interactions synchronize back to the DOT code window by executing graph update operations programmatically via python's `pydot` library.

---

## Features

- **Live Code Editor**: Write standard Graphviz DOT language with real-time compilation and syntax error warnings.
- **Drag-and-Drop Node Palette**: Drag pre-styled shape templates (Rectangle, Ellipse, Circle, Diamond, Database) and drop them directly onto the canvas.
- **Interactive Connections**: Connect nodes by dragging a path from one node to another.
- **Node Relocation**: Interactively drag nodes to customize graph layouts (when using `neato` or `fdp` layout engines).
- **Properties Inspector**: Edit labels, shapes (ellipse, box, circle, diamond, cylinder, note, etc.), fill/border colors, and edge styles (solid, dashed, dotted, bold) dynamically.
- **Infinite Canvas**: Drag to pan, scroll to zoom, and fit-to-screen controls.
- **Exporting**: One-click download of the generated SVG and copy-to-clipboard for the DOT code.

---

## Setup & Run Instructions

### Prerequisites

- **Python**: version 3.8+ (tested on Python 3.12)
- **Graphviz**: The system `dot` command-line utility must be installed and in your environment PATH.
  - On Debian/Ubuntu: `sudo apt-get install graphviz`
  - On macOS: `brew install graphviz`

### Installation

1. Navigate to the project directory:

   ```bash
   cd dynamic-graphviz-svg-example/
   ```

2. Create a virtual environment and activate it:

   ```bash
   python3 -m venv venv
   source venv/bin/activate
   ```

3. Install the dependencies:

   ```bash
   pip install -r requirements.txt
   ```

### Running the Server

Start the FastAPI server:

```bash
python main.py
```

Or run directly via uvicorn:

```bash
uvicorn main:app --host 0.0.0.0 --port 60002 --reload
```

Open your browser and navigate to `http://localhost:60002`.

---

## API Endpoints

The FastAPI backend exposes the following endpoints:

### 1. `POST /api/render`

Compiles raw DOT code and returns SVG markup and layout coordinates.

- **Request Body**:

  ```json
  {
    "dot_code": "digraph G { A -> B; }",
    "engine": "dot"
  }
  ```

- **Response**:

  ```json
  {
    "success": true,
    "svg": "<svg ...>...</svg>",
    "graph_data": { ... } // JSON layout data from Graphviz
  }
  ```

### 2. `POST /api/update`

Performs programmatic edits on a DOT graph (via `pydot`) and re-compiles the results.

- **Request Body**:

  ```json
  {
    "dot_code": "digraph G { A; B; }",
    "action": "add_edge",
    "engine": "dot",
    "params": {
      "source": "A",
      "target": "B",
      "attributes": {
        "label": "connected",
        "color": "#e74c3c"
      }
    }
  }
  ```

- **Supported Actions**:
  - `add_node`: Add or edit node.
  - `add_edge`: Add connection between nodes.
  - `delete_node`: Remove node and all its connected edges.
  - `delete_edge`: Remove connection between nodes.
  - `update_node`: Update node styling (label, shape, color, etc.).
  - `update_edge`: Update edge styling.
  - `update_graph`: Update global graph properties.
- **Response**:

  ```json
  {
    "success": true,
    "dot_code": "digraph G {\nA;\nB;\nA -> B [color=\"#e74c3c\", label=connected];\n}",
    "svg": "<svg ...>...</svg>",
    "graph_data": { ... }
  }
  ```

---

## Running Tests

Run the backend test suite using `pytest`:

```bash
pytest tests/
```

The test suite validates basic rendering, compilation errors, and all graph update API operations (`add_node`, `add_edge`, `delete_node`, etc.).
