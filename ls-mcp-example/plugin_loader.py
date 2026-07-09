import importlib
import inspect
import os
import sys
from plugin_interface import BasePlugin
from registry import PluginRegistry

def load_plugins_from_directory(directory: str, registry: PluginRegistry) -> None:
    """
    Dynamically discover and load plugins from subdirectories of the given directory.
    Iterates over subdirectories, finds Python modules/packages, inspects them for
    classes inheriting from BasePlugin, and registers them.
    """
    if not os.path.exists(directory):
        print(f"Plugins directory not found: {directory}", file=sys.stderr)
        return

    # Add directory to sys.path to allow dynamic imports of top-level modules
    abs_dir = os.path.abspath(directory)
    if abs_dir not in sys.path:
        sys.path.insert(0, abs_dir)

    for item in os.listdir(directory):
        item_path = os.path.join(directory, item)
        # Skip hidden directories, venv, or standard cache dirs
        if os.path.isdir(item_path) and not item.startswith("__") and not item.startswith("."):
            has_init = os.path.exists(os.path.join(item_path, "__init__.py"))
            has_plugin = os.path.exists(os.path.join(item_path, "plugin.py"))
            
            if has_init or has_plugin:
                # Add item_path to sys.path if it contains plugin.py directly and isn't a package
                if has_plugin and not has_init:
                    if os.path.abspath(item_path) not in sys.path:
                        sys.path.insert(0, os.path.abspath(item_path))
                
                module_name = item
                try:
                    # Clear from sys.modules if already loaded to allow reloading
                    if module_name in sys.modules:
                        del sys.modules[module_name]
                        
                    module = importlib.import_module(module_name)
                    
                    # Search classes in the module
                    classes_found = False
                    for class_name, obj in inspect.getmembers(module, inspect.isclass):
                        # Verify class is a subclass of BasePlugin and is defined in the module
                        if issubclass(obj, BasePlugin) and obj is not BasePlugin:
                            # Instantiate the class
                            plugin_instance = obj()
                            # Register it
                            registry.register(plugin_instance)
                            classes_found = True
                            print(f"Successfully registered plugin: {plugin_instance.name} (from {class_name})", file=sys.stderr)
                    
                    if not classes_found:
                        print(f"No BasePlugin classes found in module: {module_name}", file=sys.stderr)
                except Exception as e:
                    print(f"Error loading plugin package/module '{module_name}': {e}", file=sys.stderr)
