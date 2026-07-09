from abc import ABC, abstractmethod
from typing import Dict, Any

class BasePlugin(ABC):
    """
    Abstract base class defining the interface that all command plugins must implement.
    This interface enables easy integration of external commands (like ls, cat, stat, file)
    into the broader MCP server.
    """

    @property
    @abstractmethod
    def name(self) -> str:
        """
        The unique name of the plugin. This name will be exposed as the MCP tool name.
        Example: "ls", "cat", "stat"
        """
        pass

    @property
    @abstractmethod
    def description(self) -> str:
        """
        A human-readable description of what the tool does, used by the client/LLM.
        """
        pass

    @property
    @abstractmethod
    def input_schema(self) -> Dict[str, Any]:
        """
        The JSON Schema defining the input parameters that the tool expects.
        Should follow the JSON Schema format (e.g., {"type": "object", ...}).
        """
        pass

    @abstractmethod
    async def execute(self, arguments: Dict[str, Any]) -> str:
        """
        Executes the command with the provided arguments and returns the result as a string.
        
        Args:
            arguments: A dictionary containing parameter names and their values.
            
        Returns:
            The string output of the execution (e.g., stdout of the command, or structured text).
            
        Raises:
            Exception: If execution fails. The server will catch this and report it as a tool error.
        """
        pass
