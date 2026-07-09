from typing import Dict, List, Optional
from plugin_interface import BasePlugin

class PluginRegistry:
    """
    Manages the lifecycle of command plugins.
    Implements a CRUDL (Create, Read, Update, Delete, List) interface for plugins.
    """

    def __init__(self):
        self._plugins: Dict[str, BasePlugin] = {}

    def register(self, plugin: BasePlugin) -> None:
        """
        Create/Register a new plugin.
        Raises ValueError if the plugin is invalid or already registered.
        """
        if not isinstance(plugin, BasePlugin):
            raise TypeError("Plugin must inherit from BasePlugin")
        if not plugin.name:
            raise ValueError("Plugin name cannot be empty")
        if plugin.name in self._plugins:
            raise ValueError(f"Plugin with name '{plugin.name}' is already registered")
        self._plugins[plugin.name] = plugin

    def get(self, name: str) -> Optional[BasePlugin]:
        """
        Read/Retrieve a registered plugin by name.
        """
        return self._plugins.get(name)

    def update(self, name: str, plugin: BasePlugin) -> None:
        """
        Update an existing plugin registration.
        Raises KeyError if the plugin does not exist.
        """
        if name not in self._plugins:
            raise KeyError(f"Plugin '{name}' not found in registry")
        if not isinstance(plugin, BasePlugin):
            raise TypeError("Plugin must inherit from BasePlugin")
        if plugin.name != name:
            raise ValueError(f"Cannot update plugin name from '{name}' to '{plugin.name}'")
        self._plugins[name] = plugin

    def unregister(self, name: str) -> Optional[BasePlugin]:
        """
        Delete/Unregister a plugin by name. Returns the removed plugin, or None if not found.
        """
        return self._plugins.pop(name, None)

    def list(self) -> List[BasePlugin]:
        """
        List all currently registered plugins.
        """
        return list(self._plugins.values())
