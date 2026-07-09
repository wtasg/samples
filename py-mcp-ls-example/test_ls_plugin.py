import unittest
import os
import tempfile
import shutil
from plugins.ls.plugin import LsPlugin

class TestLsPlugin(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self):
        self.plugin = LsPlugin()
        # Create a temporary directory structure for testing ls
        self.test_dir = tempfile.mkdtemp()
        self.file1 = os.path.join(self.test_dir, "file1.txt")
        self.file2 = os.path.join(self.test_dir, "file2.txt")
        self.hidden_file = os.path.join(self.test_dir, ".hidden.txt")
        
        with open(self.file1, "w") as f:
            f.write("hello")
        with open(self.file2, "w") as f:
            f.write("world")
        with open(self.hidden_file, "w") as f:
            f.write("hidden content")

    async def asyncTearDown(self):
        shutil.rmtree(self.test_dir)

    def test_plugin_properties(self):
        self.assertEqual(self.plugin.name, "ls")
        self.assertIn("List directory contents", self.plugin.description)
        self.assertEqual(self.plugin.input_schema["type"], "object")

    async def test_execute_default_listing(self):
        result = await self.plugin.execute({"path": self.test_dir})
        self.assertIn("file1.txt", result)
        self.assertIn("file2.txt", result)
        self.assertNotIn(".hidden.txt", result)

    async def test_execute_show_all(self):
        result = await self.plugin.execute({"path": self.test_dir, "show_all": True})
        self.assertIn("file1.txt", result)
        self.assertIn("file2.txt", result)
        self.assertIn(".hidden.txt", result)

    async def test_execute_detailed(self):
        result = await self.plugin.execute({"path": self.test_dir, "detailed": True})
        self.assertIn("file1.txt", result)
        self.assertIn("file2.txt", result)
        # Check that detailed mode prints file size/details
        lines = result.splitlines()
        self.assertTrue(any("file1.txt" in line for line in lines))

    async def test_execute_invalid_path_raises_runtime_error(self):
        invalid_path = os.path.join(self.test_dir, "non_existent_sub_dir")
        with self.assertRaises(RuntimeError):
            await self.plugin.execute({"path": invalid_path})

if __name__ == "__main__":
    unittest.main()
