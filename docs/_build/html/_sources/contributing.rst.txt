Contributing
============

Thank you for your interest in contributing to PT! This project welcomes contributions from the community.

Development Setup
-----------------

1. Fork the repository::

   git clone https://github.com/cumulus13/pt-go.git
   cd pt-go

2. Install dependencies::

   go mod download

3. Run in development::

   go run pt/main.go --help

4. Build::

   go build -o pt pt/main.go

5. Format code::

   go fmt ./...

6. Run linters (optional)::

   golangci-lint run

Making Changes
--------------

1. Create a feature branch::

   git checkout -b feature/amazing-feature

2. Make your changes
3. Test thoroughly (see Testing section)
4. Commit your changes::

   git commit -m 'Add amazing feature'

5. Push to your fork::

   git push origin feature/amazing-feature

6. Open a Pull Request on GitHub

Code Style
----------

- Follow standard Go conventions
- Use ``go fmt`` to format code
- Add comments for exported functions
- Keep functions small and focused
- Use meaningful variable names

Testing
-------

Manual Testing
~~~~~~~~~~~~~~

Test basic functionality::

   # Test basic write
   echo "Hello World" | pbcopy  # macOS
   pt test.txt

   # Test append
   echo "Line 2" | pbcopy
   pt + test.txt

   # Test backup listing
   pt -l test.txt

   # Test restore
   pt -r test.txt --last

   # Test diff
   pt -d test.txt --last

   # Test recursive search
   mkdir -p deep/nested/path
   echo "test" | pbcopy
   pt deep/nested/path/file.txt
   cd ../../..
   pt -l file.txt  # Should find it automatically

   # Test tree
   pt -t

   # Test safe delete
   pt -rm test.txt

   # Test check command
   pt check

   # Test commit
   pt commit -m "Test commit"

Run Tests (if implemented)::
   
   go test ./...

Feature Ideas
-------------

Want to contribute? Here are some ideas:

Completed Features
~~~~~~~~~~~~~~~~~~

- ✅ Config file support (`.ptrc`) - v1.0.19
- ✅ Backup comments/metadata - v1.0.19
- ✅ Check mode to skip identical content - v1.0.19
- ✅ Recursive file search - v1.0.25
- ✅ Delta diff integration - v1.0.25
- ✅ Interactive file selection - v1.0.25

Open Ideas
~~~~~~~~~~

- Custom backup directory location (absolute path)
- Backup compression (gzip)
- Backup to cloud storage (S3, GCS)
- Web UI for backup management
- Backup cleanup strategies (by age, size)
- File watching mode (auto-backup on change)
- Backup tags (additional metadata)
- Multi-file operations
- Backup encryption

Reporting Issues
----------------

- **Bug Reports**: https://github.com/cumulus13/pt-go/issues
- **Feature Requests**: https://github.com/cumulus13/pt-go/discussions

Please include:

- PT version (``pt --version``)
- OS and version
- Steps to reproduce
- Expected vs actual behavior
- Relevant debug output (run with ``--debug``)

Pull Request Process
--------------------

1. Ensure your code follows the style guidelines
2. Update documentation if needed
3. Add tests for new features
4. Ensure all tests pass
5. Update the CHANGELOG.md (if exists)
6. Submit PR with clear description

License
-------

By contributing, you agree that your contributions will be licensed under the MIT License.