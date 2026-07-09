import unittest
from typing import Dict, Any
from plugin_interface import BasePlugin
from registry import PluginRegistry

class MockPlugin(BasePlugin):
    def __init__(self, name: str, description: str = "mock desc"):
        self._name = name
        self._description = description

    @property
    def name(self) -> str:
        return self._name

    @property
    def description(self) -> str:
        return self._description

    @property
    def input_schema(self) -> Dict[str, Any]:
        return {"type": "object"}

    async def execute(self, arguments: Dict[str, Any]) -> str:
        return "mock result"

class TestPluginRegistry(unittest.TestCase):
    def setUp(self):
        self.registry = PluginRegistry()
        self.plugin_ls = MockPlugin("ls", "list directory")
        self.plugin_cat = MockPlugin("cat", "show file content")

    def test_register_and_get(self):
        # Create
        self.registry.register(self.plugin_ls)
        # Read
        retrieved = self.registry.get("ls")
        self.assertEqual(retrieved, self.plugin_ls)

    def test_register_duplicate_raises_value_error(self):
        self.registry.register(self.plugin_ls)
        with self.assertRaises(ValueError):
            self.registry.register(MockPlugin("ls", "different"))

    def test_register_invalid_type_raises_type_error(self):
        with self.assertRaises(TypeError):
            self.registry.register("not a plugin")  # type: ignore

    def test_list_plugins(self):
        # List empty
        self.assertEqual(self.registry.list(), [])
        
        # List after registration
        self.registry.register(self.plugin_ls)
        self.registry.register(self.plugin_cat)
        plugins = self.registry.list()
        self.assertEqual(len(plugins), 2)
        self.assertIn(self.plugin_ls, plugins)
        self.assertIn(self.plugin_cat, plugins)

    def test_update_plugin(self):
        self.registry.register(self.plugin_ls)
        updated_ls = MockPlugin("ls", "updated listing tool")
        
        # Update
        self.registry.update("ls", updated_ls)
        retrieved = self.registry.get("ls")
        self.assertIsNotNone(retrieved)
        self.assertEqual(retrieved.description, "updated listing tool")

    def test_update_non_existent_raises_key_error(self):
        with self.assertRaises(KeyError):
            self.registry.update("nonexistent", self.plugin_ls)

    def test_update_mismatched_name_raises_value_error(self):
        self.registry.register(self.plugin_ls)
        with self.assertRaises(ValueError):
            self.registry.update("ls", self.plugin_cat)

    def test_unregister(self):
        self.registry.register(self.plugin_ls)
        # Delete
        removed = self.registry.unregister("ls")
        self.assertEqual(removed, self.plugin_ls)
        self.assertIsNone(self.registry.get("ls"))

    def test_unregister_non_existent(self):
        removed = self.registry.unregister("nonexistent")
        self.assertIsNone(removed)

if __name__ == "__main__":
    unittest.main()
