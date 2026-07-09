import subprocess
import os
from typing import Dict, Any
from plugin_interface import BasePlugin

class LsPlugin(BasePlugin):
    @property
    def name(self) -> str:
        return "ls"

    @property
    def description(self) -> str:
        return (
            "List directory contents. Runs the system 'ls' command and returns the output. "
            "Supports customizable arguments such as detailed view (-l) and showing hidden files (-a)."
        )

    @property
    def input_schema(self) -> Dict[str, Any]:
        return {
            "type": "object",
            "properties": {
                "path": {
                    "type": "string",
                    "description": "The directory path to list. Defaults to the current directory '.'."
                },
                "detailed": {
                    "type": "boolean",
                    "description": "If true, lists files in long format with details (permissions, owner, size, date)."
                },
                "show_all": {
                    "type": "boolean",
                    "description": "If true, shows hidden files starting with '.'."
                }
            }
        }

    async def execute(self, arguments: Dict[str, Any]) -> str:
        path = arguments.get("path", ".")
        detailed = arguments.get("detailed", False)
        show_all = arguments.get("show_all", False)

        # Safely expand path (e.g. handle ~ user expansion)
        expanded_path = os.path.expanduser(path)

        cmd = ["ls"]
        if detailed:
            cmd.append("-l")
        if show_all:
            cmd.append("-a")
        
        cmd.append(expanded_path)

        try:
            # We run subprocess directly. Since we pass arguments as a list, this prevents shell injection.
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            return result.stdout
        except subprocess.CalledProcessError as e:
            # Propagate the stderr from the command to provide context on failure
            raise RuntimeError(f"Command 'ls' failed (exit code {e.returncode}): {e.stderr or e.stdout}")
        except FileNotFoundError:
            raise RuntimeError("The system command 'ls' was not found on this environment.")
