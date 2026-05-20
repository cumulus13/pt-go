.. _api:

API Documentation (Go Source)
=============================

This section outlines the structure and key components of the Go source code for |project_name|.

.. note::
   Sphinx's standard autodoc is designed for Python. Documenting Go code directly with Sphinx requires external tools or manual documentation. This section reflects the structure inferred from the provided ``pt/main.go`` file.

Main Package
------------

Configuration
^^^^^^^^^^^^^

.. code-block:: go

   // Config holds the application configuration loaded from ``pt.yml``.
   type Config struct {
       MaxClipboardSize int    `yaml:"max_clipboard_size"`
       MaxBackupCount   int    `yaml:"max_backup_count"`
       MaxFilenameLen   int    `yaml:"max_filename_length"`
       BackupDirName    string `yaml:"backup_dir_name"`
       MaxSearchDepth   int    `yaml:"max_search_depth"`
   }

.. code-block:: go

   // Global instance of the application configuration.
   var appConfig *Config

Constants
^^^^^^^^^

.. code-block:: go

   // Default configuration constants.
   const (
       DefaultMaxClipboardSize = 100 * 1024 * 1024 // 100MB max
       DefaultMaxBackupCount   = 100                // Keep max 100 backups
       DefaultMaxFilenameLen   = 200                // Max filename length
       DefaultBackupDirName    = ".pt"              // Git-like hidden directory
       DefaultMaxSearchDepth   = 10                 // Max directory depth for recursive search
   )

.. code-block:: go

   // The version string loaded from the ``VERSION`` file.
   var Version string = "dev"

Logging
^^^^^^^

.. code-block:: go

   // Global logger instance for audit trails.
   var logger *log.Logger

Initialization
^^^^^^^^^^^^^^

.. code-block:: go

   // init initializes the global logger, version, and configuration.
   func init()

Core Functions
^^^^^^^^^^^^^^

.. code-block:: go

   // main is the entry point of the application. Parses command-line arguments and dispatches to appropriate handlers.
   func main()

.. code-block:: go

   // Command handlers for the various ``pt`` subcommands.
   func handleCheckCommand(args []string) error
   func handleCommitCommand(args []string) error
   func handleTreeCommand(args []string) error
   func handleConfigCommand(args []string) error
   func handleRemoveCommand(args []string) error
   func handleDiffCommand(args []string) error

File Operations
^^^^^^^^^^^^^^^

.. code-block:: go

   // Functions for writing, backing up, listing, and restoring files.
   func writeFile(filePath string, data string, appendMode bool, checkMode bool, comment string) error
   func autoRenameIfExists(filePath, comment string) (string, error)
   func listBackups(filePath string) ([]BackupInfo, error)
   func restoreBackup(backupPath, originalPath, comment string) error

File Status and Tree Building
^^^^^^^^^^^^^^^^^^^^^^^^^^^^^

.. code-block:: go

   // Holds status information for a file/directory node.
   type FileStatusInfo struct {
       Path     string
       RelPath  string
       Status   FileStatus
       Size     int64
       ModTime  time.Time
       IsDir    bool
       Children []*FileStatusInfo
   }

.. code-block:: go

   // Represents a node in the directory tree.
   type TreeNode struct {
       Name     string
       Path     string
       IsDir    bool
       Size     int64
       Children []*TreeNode
   }

.. code-block:: go

   // Functions to recursively build status and directory trees.
   func buildStatusTree(path string, gitignore *GitIgnore, exceptions map[string]bool, depth int, maxDepth int) (*FileStatusInfo, error)
   func buildTree(path string, gitignore *GitIgnore, exceptions map[string]bool, depth int, maxDepth int) (*TreeNode, error)

Ignore Patterns
^^^^^^^^^^^^^^^

.. code-block:: go

   // Holds ignore patterns and provides matching logic.
   type GitIgnore struct {
       patterns []string
   }

.. code-block:: go

   // Functions for loading and applying ``.gitignore`` patterns.
   func loadGitIgnore(rootPath string) (*GitIgnore, error)
   func (gi *GitIgnore) shouldIgnore(path string, isDir bool) bool

Utility Functions
^^^^^^^^^^^^^^^^^

.. code-block:: go

   // Various utility functions for loading, path finding, formatting, and printing help/version.
   func loadVersion() string
   func loadConfig() *Config
   func findPTRoot(startPath string) (string, error)
   func ensurePTDir(filePath string) (string, error)
   func resolveFilePath(filename string) (string, error)
   func formatSize(size int64) string
   func printHelp()
   func printVersion()