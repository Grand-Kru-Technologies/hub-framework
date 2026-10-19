import unittest
from unittest.mock import patch, MagicMock
import os
from utils import find_project_root
from start import server_options, start

class TestStart(unittest.TestCase):
    def setUp(self):
        self.project_root = find_project_root()
        os.chdir(self.project_root)

    @patch('start.DynamicHandlerImporter')
    @patch('start.create_app')
    def test_start(self, mock_create_app, mock_importer):
        # Mock the importer and its load_handlers method
        mock_importer_instance = MagicMock()
        mock_importer_instance.load_handlers.return_value = {'test_handler': lambda x: x}
        mock_importer.return_value = mock_importer_instance

        # Mock create_app
        mock_app = MagicMock()
        mock_create_app.return_value = mock_app

        # Call the start function
        yaml_file_path = 'path/to/test_handlers.yaml'
        result = start(yaml_file_path)

        # Assert that DynamicHandlerImporter was called with the correct file path
        mock_importer.assert_called_once_with(yaml_file_path)

        # Assert that load_handlers was called
        mock_importer_instance.load_handlers.assert_called_once()

        # Assert that create_app was called with a dictionary containing 'test_handler'
        mock_create_app.assert_called_once()
        handlers_arg = mock_create_app.call_args[0][0]
        self.assertIn('test_handler', handlers_arg)
        self.assertTrue(callable(handlers_arg['test_handler']))

        # Assert that the function returns the app instance
        self.assertEqual(result, mock_app)

    def test_server_options_default_to_loopback_without_debug(self):
        self.assertEqual(
            server_options({}),
            {'host': '127.0.0.1', 'port': 5000, 'debug': False},
        )

    def test_server_options_use_explicit_container_settings(self):
        self.assertEqual(
            server_options(
                {'HOST': '192.0.2.10', 'PORT': '5100', 'FLASK_DEBUG': '0'}
            ),
            {'host': '192.0.2.10', 'port': 5100, 'debug': False},
        )

    def test_server_options_allow_explicit_debug_opt_in(self):
        self.assertTrue(server_options({'FLASK_DEBUG': '1'})['debug'])

if __name__ == '__main__':
    unittest.main()
