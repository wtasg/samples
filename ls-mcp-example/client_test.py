import asyncio
import sys
from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

async def run_client():
    # Spawns mcp_server.py using the current python executable (the venv)
    server_params = StdioServerParameters(
        command=sys.executable,
        args=["mcp_server.py"]
    )
    
    print("Connecting to MCP server...")
    async with stdio_client(server_params) as (read, write):
        async with ClientSession(read, write) as session:
            # Initialize connection
            await session.initialize()
            print("Connection initialized.")
            
            # 1. List tools
            print("\n--- Discovery: Listing Tools ---")
            tools_result = await session.list_tools()
            for tool in tools_result.tools:
                print(f"Tool Name: {tool.name}")
                print(f"Description: {tool.description}")
                print(f"Schema: {tool.inputSchema}")
            
            # Check if ls is in tools
            tool_names = [t.name for t in tools_result.tools]
            if "ls" not in tool_names:
                print("ERROR: 'ls' tool was not found in registered tools!", file=sys.stderr)
                sys.exit(1)
            
            # 2. Execute standard ls
            print("\n--- Executing: ls (default arguments) ---")
            result = await session.call_tool("ls", arguments={})
            if hasattr(result, "isError") and result.isError:
                print(f"Error returned: {result.content[0].text}")
            else:
                print(result.content[0].text)

            # 3. Execute detailed ls
            print("\n--- Executing: ls (detailed=True) ---")
            result = await session.call_tool("ls", arguments={"detailed": True})
            if hasattr(result, "isError") and result.isError:
                print(f"Error returned: {result.content[0].text}")
            else:
                print(result.content[0].text)

            # 4. Execute ls on invalid path
            print("\n--- Executing: ls (path='/non_existent_path') ---")
            try:
                result = await session.call_tool("ls", arguments={"path": "/non_existent_path"})
                # Check if it was returned as an error or raised
                if hasattr(result, "isError") and result.isError:
                    print(f"Error returned as expected: {result.content[0].text}")
                else:
                    print(f"Server returned (unexpected success): {result.content[0].text}")
            except Exception as e:
                print(f"Exception raised as expected: {e}")

if __name__ == "__main__":
    asyncio.run(run_client())
