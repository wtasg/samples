import asyncio
import contextlib
import os
import sys
import uvicorn
from mcp.server import Server
from mcp.server.stdio import stdio_server
from mcp.server.streamable_http_manager import StreamableHTTPSessionManager
import mcp.types as types
from starlette.middleware.cors import CORSMiddleware
from starlette.responses import Response

from registry import PluginRegistry
from plugin_loader import load_plugins_from_directory

# Create the MCP Server
server = Server("command-mcp-server")

# Instantiate the plugin registry
registry = PluginRegistry()

# Load plugins from the 'plugins' directory relative to this server file
current_dir = os.path.dirname(os.path.abspath(__file__))
plugins_dir = os.path.join(current_dir, "plugins")
load_plugins_from_directory(plugins_dir, registry)

@server.list_tools()
async def handle_list_tools() -> list[types.Tool]:
    """
    List all available tools by querying the plugin registry.
    """
    tools = []
    for plugin in registry.list():
        tools.append(
            types.Tool(
                name=plugin.name,
                description=plugin.description,
                inputSchema=plugin.input_schema
            )
        )
    return tools

@server.call_tool()
async def handle_call_tool(name: str, arguments: dict) -> list[types.TextContent]:
    """
    Handle a tool execution request by routing it to the appropriate registered plugin.
    """
    plugin = registry.get(name)
    if not plugin:
        raise ValueError(f"Tool '{name}' not found")
        
    try:
        output = await plugin.execute(arguments)
        return [types.TextContent(type="text", text=output)]
    except Exception as e:
        raise RuntimeError(f"Plugin '{name}' execution failed: {str(e)}")

async def run_stdio():
    print("Starting MCP server over stdio...", file=sys.stderr)
    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            server.create_initialization_options()
        )

async def run_http():
    port = 60004
    manager = StreamableHTTPSessionManager(server)

    async def raw_app(scope, receive, send):
        if scope["type"] == "lifespan":
            while True:
                message = await receive()
                if message["type"] == "lifespan.startup":
                    try:
                        async with manager.run():
                            await send({"type": "lifespan.startup.complete"})
                            while True:
                                msg = await receive()
                                if msg["type"] == "lifespan.shutdown":
                                    await send({"type": "lifespan.shutdown.complete"})
                                    break
                    except Exception as e:
                        print(f"Lifespan error: {e}", file=sys.stderr)
                    return

        if scope["type"] == "http":
            path = scope["path"]
            if path in ("/mcp", "/mcp/"):
                await manager.handle_request(scope, receive, send)
                return
            response = Response("Not Found", status_code=404)
            await response(scope, receive, send)

    app_with_cors = CORSMiddleware(
        raw_app,
        allow_origins=["*"],
        allow_methods=["*"],
        allow_headers=["*"],
        expose_headers=["mcp-session-id", "Mcp-Session-Id", "mcp-protocol-version", "Mcp-Protocol-Version"],
    )

    config = uvicorn.Config(app_with_cors, host="0.0.0.0", port=port, log_level="info")
    uv_server = uvicorn.Server(config)
    print(f"Starting MCP server over Streamable HTTP on port {port}...", file=sys.stderr)
    await uv_server.serve()

async def main():
    if len(sys.argv) > 1 and sys.argv[1] in ("sse", "http"):
        await run_http()
    else:
        await run_stdio()

if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        print("\nMCP Server stopped by user.", file=sys.stderr)
        sys.exit(0)
