import asyncio
import sys
from mcp import ClientSession
from mcp.client.streamable_http import streamablehttp_client

async def run_client():
    url = "http://localhost:60004/mcp"
    print(f"Connecting to Streamable HTTP MCP server at {url}...")
    
    try:
        async with streamablehttp_client(url) as (read, write, get_session_id):
            async with ClientSession(read, write) as session:
                await session.initialize()
                print("Connection initialized successfully.")
                print(f"Session ID: {get_session_id()}")
                
                # 1. List tools
                tools_result = await session.list_tools()
                print("\n--- Discovery: Listing Tools ---")
                for tool in tools_result.tools:
                    print(f"Tool Name: {tool.name}")
                    print(f"Description: {tool.description}")
                
                # 2. Execute ls
                print("\n--- Executing: ls ---")
                result = await session.call_tool("ls", arguments={})
                print(result.content[0].text)
                
    except Exception as e:
        print(f"Failed to connect or execute tool over Streamable HTTP: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(run_client())
