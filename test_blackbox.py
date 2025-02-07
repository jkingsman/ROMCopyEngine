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


class ROMCopyEngineTest(unittest.TestCase):
    def setUp(self):
        """Create temporary directories for each test."""
        self.source_temp_folder, self.destination_temp_folder = self.create_temp_folders()

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
        for item in structure:
            full_path = os.path.join(base_path, item['path'])

            # Create parent directories if they don't exist
            os.makedirs(os.path.dirname(full_path), exist_ok=True)

            if item.get('is_dir', False):
                os.makedirs(full_path, exist_ok=True)
            else:
                with open(full_path, 'w') as f:
                    f.write(item.get('contents', ''))

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
            # Add empty directories
            for d in dirs:
                full_path = os.path.join(root, d)
                if not os.listdir(full_path):  # Only add if empty
                    rel_path = os.path.relpath(full_path, directory)
                    result.append({
                        'path': rel_path,
                        'is_dir': True
                    })

            # Add files
            for f in files:
                full_path = os.path.join(root, f)
                rel_path = os.path.relpath(full_path, directory)

                with open(full_path, 'r') as file:
                    try:
                        contents = file.read()
                    except UnicodeDecodeError:
                        # For binary files, we'll just note their existence without contents
                        contents = ''

                result.append({
                    'path': rel_path,
                    'contents': contents
                })

        return sorted(result, key=lambda x: x['path'])

    @staticmethod
    def normalize_structure(structure: List[Dict[str, Any]]) -> List[Dict[str, Any]]:
        """Normalize a structure for comparison by handling empty contents consistently."""
        normalized = []
        for item in structure:
            new_item = item.copy()
            if 'contents' not in new_item:
                new_item['contents'] = ''
            normalized.append(new_item)
        return sorted(normalized, key=lambda x: x['path'])

    def assertStructuresEqual(self, expected: List[Dict[str, Any]], actual: List[Dict[str, Any]], msg=None):
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
            diff = ''.join(difflib.unified_diff(
                expected_json.splitlines(keepends=True),
                actual_json.splitlines(keepends=True),
                fromfile='expected',
                tofile='actual'
            ))
            self.fail(f"Structures differ:\n{diff}")

    def execute_rom_copy_engine(self, source_dir: str, target_dir: str, options: str = "") -> subprocess.CompletedProcess:
        """Execute the ROMCopyEngine with given parameters.

        Args:
            source_dir: Source directory path
            target_dir: Target directory path
            options: Additional command line options as a string

        Returns:
            CompletedProcess instance with return code and output
        """
        command = [
            "go", "run", "ROMCopyEngine.go", "--skipConfirm",
            "--sourceDir", source_dir,
            "--targetDir", target_dir,
        ]

        if options:
            command.extend(options.split())

        job = subprocess.run(
            command,
            capture_output=True,
            text=True,
            cwd=os.path.dirname(os.path.abspath(__file__))
        )

        return job

    def run_copy_test(
        self,
        source_struct: List[Dict[str, Any]],
        dest_struct: List[Dict[str, Any]],
        expected_struct: List[Dict[str, Any]],
        options: str = ""
    ) -> None:
        """Run a copy test with given file structures and validate the results.

        Args:
            source_struct: Source directory structure
            dest_struct: Destination directory structure
            expected_struct: Expected destination structure after copy
            options: Additional command line options as a string
        """
        # Create initial file structures
        self.create_files_folders(self.source_temp_folder, source_struct)
        self.create_files_folders(self.destination_temp_folder, dest_struct)

        # Run the copy engine
        result = self.execute_rom_copy_engine(self.source_temp_folder, self.destination_temp_folder, options)
        self.assertEqual(result.returncode, 0, f"ROMCopyEngine failed with exit code {result.returncode}:\n{result.stderr}")

        # Get actual structure and compare
        actual_destination_file_folder_struct = self.get_files_folders(self.destination_temp_folder)
        self.assertStructuresEqual(expected_struct, actual_destination_file_folder_struct)

    # Common test structures
    BASIC_SOURCE_STRUCTURE = [
        {'path': 'snes/file1.snes'},
        {'path': 'snes/file2.snes'},
        {'path': 'snes/nested_dir/image.png'},
        {'path': 'snes/file.xml', 'contents': '<xml>foo</xml>'},
        {'path': 'nes', 'is_dir': True},
        {'path': 'psx/game1.bin'},
        {'path': 'psx/game2.bin'},
        {'path': 'psx/multidisk/game3_disk1.bin'},
        {'path': 'psx/multidisk/game3_disk2.bin'},
        {'path': 'psx/multidisk/game3.m3u', 'contents': './multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin'},
        {'path': 'psx/images/game1.png'},
        {'path': 'psx/images/game2.png'},
        {'path': 'psx/gameslist.xml', 'contents': '<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>'},
    ]

    EMPTY_DESTINATION = [
        {'path': 'snes', 'is_dir': True},
        {'path': 'PS1', 'is_dir': True},
    ]

    def test_basic_copy(self):
        """Test a basic copy operation with the example from the documentation."""
        source_file_folder_struct = [
            {'path': 'snes/file1.snes'},
            {'path': 'snes/file2.snes'},
            {'path': 'snes/nested_dir/image.png'},
            {'path': 'snes/file.xml', 'contents': '<xml>foo</xml>'},
            {'path': 'nes', 'is_dir': True},
        ]

        destination_file_folder_struct = [
            {'path': 'snes', 'is_dir': True},
        ]

        expected_destination_file_folder_struct = [
            {'path': 'snes/file1.snes'},
            {'path': 'snes/file2.snes'},
            {'path': 'snes/nested_dir/image.png'},
            {'path': 'snes/file.xml', 'contents': '<xml>foo</xml>'},
        ]

        self.run_copy_test(
            source_file_folder_struct,
            destination_file_folder_struct,
            expected_destination_file_folder_struct,
            "--mapping snes:snes --skipConfirm"
        )

    def test_basic_copy_with_stray_injected_file(self):
        """Test that stray files in the destination are preserved."""
        source_file_folder_struct = [
            {'path': 'snes/file1.snes'},
            {'path': 'snes/file2.snes'},
            {'path': 'snes/nested_dir/image.png'},
            {'path': 'snes/file.xml', 'contents': '<xml>foo</xml>'},
            {'path': 'nes', 'is_dir': True},
        ]

        destination_file_folder_struct = [
            {'path': 'snes', 'is_dir': True},
            {'path': 'snes/not_belong.snes'},
        ]

        expected_destination_file_folder_struct = [
            {'path': 'snes/file1.snes'},
            {'path': 'snes/file2.snes'},
            {'path': 'snes/nested_dir/image.png'},
            {'path': 'snes/file.xml', 'contents': '<xml>foo</xml>'},
            {'path': 'snes/not_belong.snes'},
        ]

        self.run_copy_test(
            source_file_folder_struct,
            destination_file_folder_struct,
            expected_destination_file_folder_struct,
            "--mapping snes:snes --skipConfirm"
        )

    def test_multiple_mappings(self):
        """Test that multiple platform mappings work correctly."""
        expected_structure = [
            {'path': 'PS1/game1.bin'},
            {'path': 'PS1/game2.bin'},
            {'path': 'PS1/multidisk/game3_disk1.bin'},
            {'path': 'PS1/multidisk/game3_disk2.bin'},
            {'path': 'PS1/multidisk/game3.m3u', 'contents': './multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin'},
            {'path': 'PS1/images/game1.png'},
            {'path': 'PS1/images/game2.png'},
            {'path': 'PS1/gameslist.xml', 'contents': '<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>'},
            {'path': 'SFC/file1.snes'},
            {'path': 'SFC/file2.snes'},
            {'path': 'SFC/nested_dir/image.png'},
            {'path': 'SFC/file.xml', 'contents': '<xml>foo</xml>'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            self.EMPTY_DESTINATION,
            expected_structure,
            "--mapping snes:SFC --mapping psx:PS1"
        )

    def test_copy_exclude(self):
        """Test that --copyExclude flag works correctly."""
        expected_structure = [
            {'path': 'SFC/file1.snes'},
            {'path': 'SFC/file2.snes'},
            {'path': 'SFC/file.xml', 'contents': '<xml>foo</xml>'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            self.EMPTY_DESTINATION,
            expected_structure,
            "--mapping snes:SFC --copyExclude '**/*.png'"
        )

    def test_explode_dir(self):
        """Test that --explodeDir moves files from subdirectories to parent directory."""
        expected_structure = [
            {'path': 'PS1/game1.bin'},
            {'path': 'PS1/game2.bin'},
            {'path': 'PS1/game3_disk1.bin'},
            {'path': 'PS1/game3_disk2.bin'},
            {'path': 'PS1/game3.m3u', 'contents': './game3_disk1.bin\n./game3_disk2.bin'},
            {'path': 'PS1/game1.png'},
            {'path': 'PS1/game2.png'},
            {'path': 'PS1/gameslist.xml', 'contents': '<game>\n  <path>game1.bin</path>\n  <image>./game1.png</image>\n</game>'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            self.EMPTY_DESTINATION,
            expected_structure,
            "--mapping psx:PS1 --explodeDir multidisk --explodeDir images --rewrite *.m3u:\./multidisk/:./ --rewrite *.xml:\.\./psx/images/:./ --rewritesAreRegex"
        )

    def test_rename_files(self):
        """Test that --rename flag works correctly for files."""
        expected_structure = [
            {'path': 'PS1/game1.bin'},
            {'path': 'PS1/game2.bin'},
            {'path': 'PS1/multidisk/game3_disk1.bin'},
            {'path': 'PS1/multidisk/game3_disk2.bin'},
            {'path': 'PS1/multidisk/game3.m3u', 'contents': './multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin'},
            {'path': 'PS1/images/game1.png'},
            {'path': 'PS1/images/game2.png'},
            {'path': 'PS1/miyoogamelist.xml', 'contents': '<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            self.EMPTY_DESTINATION,
            expected_structure,
            "--mapping psx:PS1 --rename gameslist.xml:miyoogamelist.xml"
        )

    def test_clean_target(self):
        """Test that --cleanTarget removes existing files in target directory."""
        destination_with_files = [
            {'path': 'PS1', 'is_dir': True},
            {'path': 'PS1/old_file.bin'},
            {'path': 'PS1/should_be_removed.txt'},
        ]

        expected_structure = [
            {'path': 'PS1/game1.bin'},
            {'path': 'PS1/game2.bin'},
            {'path': 'PS1/multidisk/game3_disk1.bin'},
            {'path': 'PS1/multidisk/game3_disk2.bin'},
            {'path': 'PS1/multidisk/game3.m3u', 'contents': './multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin'},
            {'path': 'PS1/images/game1.png'},
            {'path': 'PS1/images/game2.png'},
            {'path': 'PS1/gameslist.xml', 'contents': '<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>'},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            destination_with_files,
            expected_structure,
            "--mapping psx:PS1 --cleanTarget"
        )

    def test_copy_include(self):
        """Test that --copyInclude flag works correctly."""
        expected_structure = [
            {'path': 'PS1/images', 'is_dir': True},
            {'path': 'PS1/images/game1.png'},
            {'path': 'PS1/images/game2.png'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            self.EMPTY_DESTINATION,
            expected_structure,
            "--mapping psx:PS1 --copyInclude '**/*.png'"
        )

    def test_simple_file_rewrite(self):
        """Test file content rewriting with simple patterns."""
        source_struct = [
            {'path': 'psx/playlist.txt', 'contents': 'OLDTEXT\nOLDTEXT'},
        ]

        destination_struct = [
            {'path': 'PS1', 'is_dir': True},
        ]

        expected_struct = [
            {'path': 'PS1/playlist.txt', 'contents': 'NEWTEXT\nNEWTEXT'},
        ]

        # Create initial file structures
        self.create_files_folders(self.source_temp_folder, source_struct)
        self.create_files_folders(self.destination_temp_folder, destination_struct)

        # Run the copy engine and capture output for debugging
        result = self.execute_rom_copy_engine(
            self.source_temp_folder,
            self.destination_temp_folder,
            "--mapping psx:PS1 --rewrite *.txt:OLDTEXT:NEWTEXT"
        )

        if result.returncode != 0:
            self.fail(f"ROMCopyEngine failed with exit code {result.returncode}:\nSTDOUT:\n{result.stdout}\nSTDERR:\n{result.stderr}")

        # Get actual structure and compare
        actual_destination_file_folder_struct = self.get_files_folders(self.destination_temp_folder)
        self.assertStructuresEqual(expected_struct, actual_destination_file_folder_struct)

    def test_clean_target(self):
        """Test that --cleanTarget removes existing files in target directory."""
        destination_with_files = [
            {'path': 'PS1', 'is_dir': True},
            {'path': 'PS1/old_file.bin'},
            {'path': 'PS1/should_be_removed.txt'},
        ]

        expected_structure = [
            {'path': 'PS1/game1.bin'},
            {'path': 'PS1/game2.bin'},
            {'path': 'PS1/multidisk/game3_disk1.bin'},
            {'path': 'PS1/multidisk/game3_disk2.bin'},
            {'path': 'PS1/multidisk/game3.m3u', 'contents': './multidisk/game3_disk1.bin\n./multidisk/game3_disk2.bin'},
            {'path': 'PS1/images/game1.png'},
            {'path': 'PS1/images/game2.png'},
            {'path': 'PS1/gameslist.xml', 'contents': '<game>\n  <path>game1.bin</path>\n  <image>../psx/images/game1.png</image>\n</game>'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            destination_with_files,
            expected_structure,
            "--mapping psx:PS1 --cleanTarget"
        )

    def test_copy_include(self):
        """Test that --copyInclude flag works correctly."""
        expected_structure = [
            {'path': 'PS1/images', 'is_dir': True},
            {'path': 'PS1/images/game1.png'},
            {'path': 'PS1/images/game2.png'},
            {'path': 'snes', 'is_dir': True},
        ]

        self.run_copy_test(
            self.BASIC_SOURCE_STRUCTURE,
            self.EMPTY_DESTINATION,
            expected_structure,
            "--mapping psx:PS1 --copyInclude '**/*.png'"
        )

    def test_file_rewrite(self):
        """Test that file content rewriting works correctly."""
        source_struct = [
            {'path': 'psx/playlist.m3u', 'contents': './multidisk/game1.bin\n./multidisk/game2.bin'},
            {'path': 'psx/info.xml', 'contents': '<game>\n  <image>../psx/images/game.png</image>\n</game>'},
        ]

        destination_struct = [
            {'path': 'PS1', 'is_dir': True},
        ]

        expected_struct = [
            {'path': 'PS1/playlist.m3u', 'contents': './game1.bin\n./game2.bin'},
            {'path': 'PS1/info.xml', 'contents': '<game>\n  <image>./game.png</image>\n</game>'},
        ]

        self.run_copy_test(
            source_struct,
            destination_struct,
            expected_struct,
            "--mapping psx:PS1 --rewrite *.m3u:\./multidisk/:./ --rewrite *.xml:\.\./psx/images/:./ --rewritesAreRegex"
        )

    def test_simple_file_rewrite(self):
        """Test file content rewriting with simple patterns."""

        source_struct = [
            {'path': 'psx/playlist.txt', 'contents': 'OLDTEXT\nOLDTEXT'},
        ]

        destination_struct = [
            {'path': 'PS1', 'is_dir': True},
        ]

        expected_struct = [
            {'path': 'PS1/playlist.txt', 'contents': 'NEWTEXT\nNEWTEXT'},
        ]

        self.run_copy_test(
            source_struct,
            destination_struct,
            expected_struct,
           "--mapping psx:PS1 --rewrite *.txt:OLDTEXT:NEWTEXT"
        )

if __name__ == '__main__':
    unittest.main()
