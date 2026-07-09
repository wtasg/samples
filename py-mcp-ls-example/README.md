# MCP Command Server with Plugin Registry

This is a Model Context Protocol (MCP) server written in Python that allows running shell command integrations as dynamic plugins. It features a central **CRUDL (Create, Read, Update, Delete, List) Plugin Registry** and a plugin interface.

Currently, it implements the `ls` plugin in the `plugins/ls/` folder.

## Architecture

The server dynamically loads plugins from the `plugins/` directory. Each plugin defines its own name, description, JSON input schema, and execution logic.

```text
                  ┌──────────────────────┐
                  │      MCP Client      │
                  └──────────┬───────────┘
                             │ stdio (JSON-RPC)
                             ▼
                  ┌──────────────────────┐
                  │     mcp_server.py    │
                  └──────────┬───────────┘
                             │
                             ▼
                  ┌──────────────────────┐
                  │    PluginRegistry    │◄──── (Dynamic Loader)
                  │       (CRUDL)        │
                  └──────┬────────┬──────┘
                         │        │
                         ▼        ▼
                    ┌─────────┐  ┌─────────┐
                    │LsPlugin │  │Future...│
                    └─────────┘  └─────────┘
```

- **`plugin_interface.py`**: Defines the `BasePlugin` abstract base class.
- **`registry.py`**: Implements the `PluginRegistry` CRUDL controller.
- **`plugin_loader.py`**: Discovers, imports, and registers plugins from subfolders under `plugins/`.
- **`mcp_server.py`**: Launches the MCP server, binds registry tools, and communicates over `stdio`.

---

## Setup & Installation

1. Make sure you have python 3 installed.
2. Initialize virtual environment and install dependencies:

   ```bash
   python3 -m venv .venv
   .venv/bin/pip install -r requirements.txt
   ```

---

## Running the Server

This server supports two transport modes: **Standard Input/Output (stdio)** and **Streamable HTTP**.

### 1. Run over Standard I/O (stdio)

By default, the server runs over stdio. This is suitable for local integrations like desktop applications.

```bash
.venv/bin/python3 mcp_server.py
```

### 2. Run over Streamable HTTP (Web Server)

To run the server as a persistent web service using the modern Streamable HTTP protocol, run:

```bash
.venv/bin/python3 mcp_server.py http
```

This will start a `uvicorn` web server listening on port **`60004`**.

- **Unified MCP Endpoint**: `http://localhost:60004/mcp` (handles GET, POST, and DELETE requests for bidirectional communication).

### Comparison of Transports: `stdio` vs `Streamable HTTP`

| Feature | Standard I/O (`stdio`) | Streamable HTTP |
| :--- | :--- | :--- |
| **Communication Channel** | Uses `stdin` and `stdout` of the running process. | Uses HTTP (streamable GET for event source + POST + DELETE requests). |
| **Endpoint Architecture** | N/A (direct subprocess pipeline). | Single unified endpoint (`/mcp`) for all request/response methods. |
| **Execution Context** | Must be spawned as a subprocess by the client. | Runs as a standalone persistent network service on a port. |
| **Accessibility** | Restricted to the local host machine. | Network-accessible (supports remote/cloud connections). |
| **Lifecycle** | Tied directly to the parent editor/client process. | Independent daemon (starts/stops independently of clients). |
| **Logging** | Output (e.g. `print()`) must be sent strictly to `stderr`. | Output/logs can run on `stdout` without breaking protocol JSON-RPC. |

---

## Running Tests

### 1. Unit Tests (Registry CRUDL & Plugin Logic)

Run the registry CRUDL unit tests:
```bash
.venv/bin/python3 test_registry.py
```

Run the `ls` plugin behavior unit tests:
```bash
.venv/bin/python3 test_ls_plugin.py
```

### 2. Integration Tests (End-to-End MCP Server & Client)

Run the integration client testing the local **stdio transport**:
```bash
.venv/bin/python3 client_test.py
```

Run the integration client testing the network-accessible **Streamable HTTP transport** (ensure you start the server via `mcp_server.py http` first in another terminal):
```bash
.venv/bin/python3 client_test_http.py
```

---

## Exposing More Plugins (e.g. `cat`, `stat`, `file`)

To add a new tool integration, create a folder under `plugins/` (e.g., `plugins/cat/`) with:

1. **`plugins/cat/__init__.py`**:

   ```python
   from .plugin import CatPlugin
   ```

2. **`plugins/cat/plugin.py`**:

   ```python
   import subprocess
   from typing import Dict, Any
   from plugin_interface import BasePlugin

   class CatPlugin(BasePlugin):
       @property
       def name(self) -> str:
           return "cat"

       @property
       def description(self) -> str:
           return "Read file contents safely."

       @property
       def input_schema(self) -> Dict[str, Any]:
           return {
               "type": "object",
               "properties": {
                   "path": {
                       "type": "string",
                       "description": "Path to the file to read."
                   }
               },
               "required": ["path"]
           }

       async def execute(self, arguments: Dict[str, Any]) -> str:
           path = arguments.get("path")
           try:
               result = subprocess.run(["cat", path], capture_output=True, text=True, check=True)
               return result.stdout
           except subprocess.CalledProcessError as e:
               raise RuntimeError(f"cat failed: {e.stderr or e.stdout}")
   ```

The server dynamically scans the `plugins/` directory and registers the `cat` command automatically upon restart.

---

## Client Configuration (e.g. Claude Desktop)

To connect this server to Claude Desktop, add it to your `claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "command-mcp-server": {
      "command": "/absolute/path/to/py-mcp-ls-example/.venv/bin/python3",
      "args": ["/absolute/path/to/py-mcp-ls-example/mcp_server.py"]
    }
  }
}
```
