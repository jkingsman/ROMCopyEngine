#!/usr/bin/env python3

import os
import tempfile
import shutil
import subprocess
import json
import unittest
from typing import List, Dict, Any
from dataclasses import dataclass
import difflib


@dataclass
class FileStructure:
    path: str
    contents: str = ""
    is_dir: bool = False


@dataclass
class TestFixture:
    """Test fixture for ROMCopyEngine tests containing source, destination and expected structures."""
    source_struct: List[Dict[str, Any]]
    dest_struct: List[Dict[str, Any]]
    expected_struct: List[Dict[str, Any]]
    options: str = ""


class ROMCopyEngineTest(unittest.TestCase):
    def setUp(self):
        """Create temporary directories for each test."""
        self.source_temp_folder, self.destination_temp_folder = (
            self.create_temp_folders()
        )

    def tearDown(self):
        """Clean up temporary directories after each test."""
        shutil.rmtree(self.source_temp_folder, ignore_errors=True)
        shutil.rmtree(self.destination_temp_folder, ignore_errors=True)

    @staticmethod
    def create_temp_folders() -> tuple[str, str]:
        """Create two temporary folders for source and destination."""
        source = tempfile.mkdtemp()
        destination = tempfile.mkdtemp()
        return source, destination

    @staticmethod
    def create_files_folders(base_path: str, structure: List[Dict[str, Any]]) -> None:
        """Create a directory and file structure based on the provided specification.

        Args:
            base_path: The root directory to create the structure in
            structure: List of dictionaries specifying files and folders to create
        """
        # First create all directories to ensure parent directories exist
        for item in structure:
            full_path = os.path.join(base_path, item["path"])
            if item.get("is_dir", False):
                os.makedirs(full_path, exist_ok=True)
            else:
                os.makedirs(os.path.dirname(full_path), exist_ok=True)

        # Then create all files
        for item in structure:
            full_path = os.path.join(base_path, item["path"])
            if not item.get("is_dir", False):
                with open(full_path, "w") as f:
                    f.write(item.get("contents", ""))

    @staticmethod
    def get_files_folders(directory: str) -> List[Dict[str, Any]]:
        """Walk a directory and return its structure in our standard format.

        Args:
            directory: The directory to analyze

        Returns:
            List of dictionaries describing the structure
        """
        result = []

        for root, dirs, files in os.walk(directory):
            # Add all directories (empty or not)
            for d in dirs:
                full_path = os.path.join(root, d)
                rel_path = os.path.relpath(full_path, directory)
                result.append({"path": rel_path, "is_dir": True})

            # Add files
            for f in files:
                full_path = os.path.join(root, f)
                rel_path = os.path.relpath(full_path, directory)

                with open(full_path, "r") as file:
                    try:
                        contents = file.read()
                    except UnicodeDecodeError:
                        # For binary files, we'll just note their existence without contents
                        contents = ""

                result.append({"path": rel_path, "contents": contents})

        return sorted(result, key=lambda x: x["path"])

    @staticmethod
    def normalize_structure(structure: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Normalize a structure for comparison by handling empty contents consistently."""
        normalized = []
        for item in structure:
            new_item = item.copy()
            if "contents" not in new_item:
                new_item["contents"] = ""
            normalized.append(new_item)
        return sorted(normalized, key=lambda x: x["path"])

    def assertStructuresEqual(
        self, expected: List[Dict[str, Any]], actual: List[Dict[str, Any]], msg=None
    ):
        """Assert that two directory structures are equal and show differences if not.

        Args:
            expected: Expected directory structure
            actual: Actual directory structure
            msg: Optional message to display on failure
        """
        normalized_expected = self.normalize_structure(expected)
        normalized_actual = self.normalize_structure(actual)

        expected_json = json.dumps(normalized_expected, sort_keys=True, indent=2)
        actual_json = json.dumps(normalized_actual, sort_keys=True, indent=2)

        if expected_json != actual_json:
            diff = "".join(
                difflib.unified_diff(
                    expected_json.splitlines(keepends=True),
                    actual_json.splitlines(keepends=True),
                    fromfile="expected",
                    tofile="actual",
                )
            )
            self.fail(f"Structures differ:\n{diff}")

    def execute_rom_copy_engine(
        self, source_dir: str, target_dir: str, options: str = ""
    ) -> subprocess.CompletedProcess:
        """Execute the ROMCopyEngine with given parameters.

        Args:
            source_dir: Source directory path
            target_dir: Target directory path
            options: Additional command line options as a string

        Returns:
            CompletedProcess instance with return code and output
        """
        command = [
            "go",
            "run",
            "ROMCopyEngine.go",
            "--skipConfirm",
            "--sourceDir",
            source_dir,
            "--targetDir",
            target_dir,
        ]

        if options:
            command.extend(options.split())

        job = subprocess.run(
            ' '.join(command),
            capture_output=True,
            shell=True,
            text=True,
            cwd=os.path.dirname(os.path.abspath(__file__)),
        )

        # print(' '.join(command))
        # print(job.stdout)

        # import time
        # time.sleep(9999999)
        return job

    def run_copy_test(self, fixture: TestFixture) -> None:
        """Run a copy test with given test fixture.

        Args:
            fixture: TestFixture containing source, destination, expected structures and options
        """
        # Create initial file structures
        self.create_files_folders(self.source_temp_folder, fixture.source_struct)
        self.create_files_folders(self.destination_temp_folder, fixture.dest_struct)

        # Run the copy engine
        result = self.execute_rom_copy_engine(
            self.source_temp_folder, self.destination_temp_folder, fixture.options
        )
        self.assertEqual(
            result.returncode,
            0,
            f"ROMCopyEngine failed with exit code {result.returncode}:\n{result.stderr}",
        )

        # Get actual structure and compare
        actual_destination_file_folder_struct = self.get_files_folders(
            self.destination_temp_folder
        )
        self.assertStructuresEqual(
            fixture.expected_struct, actual_destination_file_folder_struct
        )

    # Common test structures
    BASIC_SOURCE_STRUCTURE = [
        {"path": "snes/file1.snes"},
        {"path": "snes/file2.snes"},
        {"path": "snes/nested_dir/image.png"},
        {"path": "snes/file.xml", "contents": "<xml>foo</xml>"},
        {"path": "nes", "is_dir": True},
        {"path": "psx/game1.bin"},
        {"path": "psx/game2.bin"},
        {"path": "psx/multidisk/game3_disk1.bin"},
        {"path": "psx/multidisk/game3_disk2.bin"},
        {
            "path": "psx/multidisk/game3.m3u",
            "contents": "./multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin",
        },
        {"path": "psx/images/game1.png"},
        {"path": "psx/images/game2.png"},
        {
            "path": "psx/gameslist.xml",
            "contents": "<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>",
        },
    ]

    EMPTY_DESTINATION = [
        {"path": "snes", "is_dir": True},
        {"path": "PS1", "is_dir": True},
    ]

    def test_basic_copy(self):
        """Test a basic copy operation with the example from the documentation."""
        fixture = TestFixture(
            source_struct=[
                {"path": "snes/file1.snes"},
                {"path": "snes/file2.snes"},
                {"path": "snes/nested_dir/image.png"},
                {"path": "snes/file.xml", "contents": "<xml>foo</xml>"},
                {"path": "nes", "is_dir": True},
            ],
            dest_struct=[{"path": "snes", "is_dir": True}],
            expected_struct=[
                {"path": "snes", "is_dir": True},
                {"path": "snes/file1.snes"},
                {"path": "snes/file2.snes"},
                {"path": "snes/file.xml", "contents": "<xml>foo</xml>"},
                {"path": "snes/nested_dir", "is_dir": True},
                {"path": "snes/nested_dir/image.png"},
            ],
            options="--mapping snes:snes --skipConfirm"
        )
        self.run_copy_test(fixture)

    def test_basic_copy_with_stray_injected_file(self):
        """Test that stray files in the destination are preserved."""
        source_file_folder_struct = [
            {"path": "snes/file1.snes"},
            {"path": "snes/file2.snes"},
            {"path": "snes/nested_dir/image.png"},
            {"path": "snes/file.xml", "contents": "<xml>foo</xml>"},
            {"path": "nes", "is_dir": True},
        ]

        destination_file_folder_struct = [
            {"path": "snes", "is_dir": True},
            {"path": "snes/not_belong.snes"},
        ]

        expected_destination_file_folder_struct = [
            {"path": "snes", "is_dir": True},
            {"path": "snes/file1.snes"},
            {"path": "snes/file2.snes"},
            {"path": "snes/file.xml", "contents": "<xml>foo</xml>"},
            {"path": "snes/nested_dir", "is_dir": True},
            {"path": "snes/nested_dir/image.png"},
            {"path": "snes/not_belong.snes"},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=source_file_folder_struct,
                dest_struct=destination_file_folder_struct,
                expected_struct=expected_destination_file_folder_struct,
                options="--mapping snes:snes --skipConfirm",
            )
        )

    def test_multiple_mappings(self):
        """Test that multiple platform mappings work correctly."""
        expected_structure = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/game1.bin"},
            {"path": "PS1/game2.bin"},
            {"path": "PS1/images", "is_dir": True},
            {"path": "PS1/images/game1.png"},
            {"path": "PS1/images/game2.png"},
            {"path": "PS1/multidisk", "is_dir": True},
            {"path": "PS1/multidisk/game3_disk1.bin"},
            {"path": "PS1/multidisk/game3_disk2.bin"},
            {
                "path": "PS1/multidisk/game3.m3u",
                "contents": "./multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin",
            },
            {
                "path": "PS1/gameslist.xml",
                "contents": "<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>",
            },
            {"path": "snes", "is_dir": True},
            {"path": "snes/file1.snes"},
            {"path": "snes/file2.snes"},
            {"path": "snes/file.xml", "contents": "<xml>foo</xml>"},
            {"path": "snes/nested_dir", "is_dir": True},
            {"path": "snes/nested_dir/image.png"},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=self.BASIC_SOURCE_STRUCTURE,
                dest_struct=self.EMPTY_DESTINATION,
                expected_struct=expected_structure,
                options="--mapping snes:snes --mapping psx:PS1",
            )
        )

    def test_copy_exclude(self):
        """Test that --copyExclude flag works correctly."""
        source_file_folder_struct = [
            {"path": "snes/file1.snes"},
            {"path": "snes/file2.snes"},
            {"path": "snes/nested_dir/image.png"},
            {"path": "snes/img.png"},
            {"path": "snes/png.not"},
            {"path": "nes", "is_dir": True},
        ]

        destination_file_folder_struct = [
            {"path": "snes", "is_dir": True},
        ]

        expected_destination_file_folder_struct = [
            {"path": "snes", "is_dir": True},
            {"path": "snes/img.png"},
            {"path": "snes/nested_dir", "is_dir": True},
            {"path": "snes/nested_dir/image.png"},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=source_file_folder_struct,
                dest_struct=destination_file_folder_struct,
                expected_struct=expected_destination_file_folder_struct,
                options="--mapping snes:snes --copyInclude **/*.png --skipConfirm",
            )
        )

    def test_explode_dir(self):
        """Test that --explodeDir moves files from subdirectories to parent directory."""
        expected_structure = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/game1.bin"},
            {"path": "PS1/game2.bin"},
            {"path": "PS1/game3_disk1.bin"},
            {"path": "PS1/game3_disk2.bin"},
            {
                "path": "PS1/game3.m3u",
                "contents": "./game3_disk1.bin\n./game3_disk2.bin",
            },
            {"path": "PS1/game1.png"},
            {"path": "PS1/game2.png"},
            {
                "path": "PS1/gameslist.xml",
                "contents": "<game>\n  <path>game1.bin</path>\n  <image>./game1.png</image>\n</game>",
            },
            {"path": "snes", "is_dir": True},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=self.BASIC_SOURCE_STRUCTURE,
                dest_struct=self.EMPTY_DESTINATION,
                expected_struct=expected_structure,
                options="--mapping psx:PS1 --explodeDir multidisk --explodeDir images --rewrite *.m3u:./multidisk/:./ --rewrite *.xml:../psx/images/:./ --rewritesAreRegex",
            )
        )

    def test_rename_files(self):
        """Test that --rename flag works correctly for files."""
        expected_structure = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/game1.bin"},
            {"path": "PS1/game2.bin"},
            {"path": "PS1/images", "is_dir": True},
            {"path": "PS1/images/game1.png"},
            {"path": "PS1/images/game2.png"},
            {"path": "PS1/multidisk", "is_dir": True},
            {"path": "PS1/multidisk/game3_disk1.bin"},
            {"path": "PS1/multidisk/game3_disk2.bin"},
            {
                "path": "PS1/multidisk/game3.m3u",
                "contents": "./multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin",
            },
            {
                "path": "PS1/miyoogamelist.xml",
                "contents": "<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>",
            },
            {"path": "snes", "is_dir": True},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=self.BASIC_SOURCE_STRUCTURE,
                dest_struct=self.EMPTY_DESTINATION,
                expected_struct=expected_structure,
                options="--mapping psx:PS1 --rename gameslist.xml:miyoogamelist.xml",
            )
        )

    def test_clean_target(self):
        """Test that --cleanTarget removes existing files in target directory."""
        destination_with_files = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/old_file.bin"},
            {"path": "PS1/should_be_removed.txt"},
        ]

        expected_structure = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/game1.bin"},
            {"path": "PS1/game2.bin"},
            {"path": "PS1/images", "is_dir": True},
            {"path": "PS1/images/game1.png"},
            {"path": "PS1/images/game2.png"},
            {"path": "PS1/multidisk", "is_dir": True},
            {"path": "PS1/multidisk/game3_disk1.bin"},
            {"path": "PS1/multidisk/game3_disk2.bin"},
            {
                "path": "PS1/multidisk/game3.m3u",
                "contents": "./multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin",
            },
            {
                "path": "PS1/gameslist.xml",
                "contents": "<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>",
            },
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=self.BASIC_SOURCE_STRUCTURE,
                dest_struct=destination_with_files,
                expected_struct=expected_structure,
                options="--mapping psx:PS1 --cleanTarget",
            )
        )

    def test_empty_directory_handling(self):
        """Test that empty directories are properly created and preserved."""
        source_struct = [
            {"path": "snes/empty1", "is_dir": True},
            {"path": "snes/empty2", "is_dir": True},
            {"path": "snes/nonempty", "is_dir": True},
            {"path": "snes/nonempty/file.txt", "contents": "test"},
            {"path": "psx/empty3", "is_dir": True},
        ]

        destination_struct = [
            {"path": "snes", "is_dir": True},
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/existing_empty", "is_dir": True},
        ]

        expected_struct = [
            {"path": "snes", "is_dir": True},
            {"path": "snes/empty1", "is_dir": True},
            {"path": "snes/empty2", "is_dir": True},
            {"path": "snes/nonempty", "is_dir": True},
            {"path": "snes/nonempty/file.txt", "contents": "test"},
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/empty3", "is_dir": True},
            {"path": "PS1/existing_empty", "is_dir": True},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=source_struct,
                dest_struct=destination_struct,
                expected_struct=expected_struct,
                options="--mapping snes:snes --mapping psx:PS1",
            )
        )

    def test_copy_include(self):
        """Test that --copyInclude flag works correctly."""
        expected_structure = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/images", "is_dir": True},
            {"path": "PS1/images/game1.png"},
            {"path": "PS1/images/game2.png"},
            {"path": "snes", "is_dir": True},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=self.BASIC_SOURCE_STRUCTURE,
                dest_struct=self.EMPTY_DESTINATION,
                expected_struct=expected_structure,
                options="--mapping psx:PS1 --copyInclude **/*.png",
            )
        )

    def test_file_rewrite(self):
        """Test that file content rewriting works correctly."""
        source_struct = [
            {
                "path": "psx/playlist.m3u",
                "contents": "./multidisk/game1.bin\n./multidisk/game2.bin",
            },
            {
                "path": "psx/info.xml",
                "contents": "<game>\n  <image>../psx/images/game.png</image>\n</game>",
            },
        ]

        destination_struct = [
            {"path": "PS1", "is_dir": True},
        ]

        expected_struct = [
            {"path": "PS1", "is_dir": True},
            {
                "path": "PS1/info.xml",
                "contents": "<game>\n  <image>./game.png</image>\n</game>",
            },
            {"path": "PS1/playlist.m3u", "contents": "./game1.bin\n./game2.bin"},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=source_struct,
                dest_struct=destination_struct,
                expected_struct=expected_struct,
                options="--mapping psx:PS1 --rewrite *.m3u:./multidisk/:./ --rewrite *.xml:../psx/images/:./ --rewritesAreRegex",
            )
        )

    def test_simple_file_rewrite(self):
        """Test file content rewriting with simple patterns."""
        source_struct = [
            {"path": "psx/playlist.txt", "contents": "OLDTEXT\nOLDTEXT"},
        ]

        destination_struct = [
            {"path": "PS1", "is_dir": True},
        ]

        expected_struct = [
            {"path": "PS1", "is_dir": True},
            {"path": "PS1/playlist.txt", "contents": "NEWTEXT\nNEWTEXT"},
        ]

        self.run_copy_test(
            TestFixture(
                source_struct=source_struct,
                dest_struct=destination_struct,
                expected_struct=expected_struct,
                options="--mapping psx:PS1 --rewrite *.txt:OLDTEXT:NEWTEXT",
            )
        )


if __name__ == "__main__":
    unittest.main()
