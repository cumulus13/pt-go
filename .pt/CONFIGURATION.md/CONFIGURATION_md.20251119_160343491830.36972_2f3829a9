# PT Configuration Guide

PT can be customized using a YAML configuration file. This allows you to adjust limits, backup behavior, and search depth according to your needs.

## Quick Start

Generate a sample config file:

```bash
pt config init
```

This creates `pt.yml` in the current directory with all available options and their default values.

## Config File Locations

PT searches for config files in the following locations (in order of priority):

1. `./pt.yml` or `./pt.yaml` (current directory)
2. `~/.config/pt/pt.yml` or `~/.config/pt/pt.yaml`
3. `~/pt.yml` or `~/pt.yaml` (home directory)
4. Hidden variants: `.pt.yml` or `.pt.yaml` in any of the above

The first file found will be used. If no config file is found, PT uses default values.

## Configuration Options

### max_clipboard_size

Maximum clipboard content size in bytes.

- **Default**: `104857600` (100 MB)
- **Range**: 1 - 1073741824 (1 GB)
- **Description**: Prevents accidentally saving huge content that could fill up disk space

```yaml
max_clipboard_size: 104857600  # 100 MB
```

Examples:
```yaml
max_clipboard_size: 52428800    # 50 MB
max_clipboard_size: 209715200   # 200 MB
max_clipboard_size: 524288000   # 500 MB
```

### max_backup_count

Maximum number of backup files to keep per file.

- **Default**: `100`
- **Range**: 1 - 10000
- **Description**: Older backups are automatically removed when this limit is reached

```yaml
max_backup_count: 100
```

Examples:
```yaml
max_backup_count: 50     # Keep last 50 versions
max_backup_count: 200    # Keep last 200 versions
max_backup_count: 1000   # Keep last 1000 versions
```

### max_filename_length

Maximum filename length in characters.

- **Default**: `200`
- **Range**: 1 - 1000
- **Description**: Prevents issues with filesystem limits

```yaml
max_filename_length: 200
```

Examples:
```yaml
max_filename_length: 100   # Shorter limit
max_filename_length: 255   # Common filesystem limit
max_filename_length: 500   # Longer filenames
```

### backup_dir_name

Name of the backup directory.

- **Default**: `backup`
- **Description**: Backups are stored in `./{backup_dir_name}/` relative to the file location

```yaml
backup_dir_name: backup
```

Examples:
```yaml
backup_dir_name: .backups      # Hidden directory
backup_dir_name: versions      # More descriptive name
backup_dir_name: pt-history    # Prefixed name
```

### max_search_depth

Maximum directory depth for recursive file search.

- **Default**: `10`
- **Range**: 1 - 100
- **Description**: How deep to search when a file is not found in current directory

```yaml
max_search_depth: 10
```

Examples:
```yaml
max_search_depth: 5     # Shallow search (faster)
max_search_depth: 20    # Deep search (slower, more thorough)
max_search_depth: 3     # Very shallow (immediate subdirectories only)
```

## Complete Example Config

```yaml
# PT Configuration File
# Custom settings for my workflow

# Allow larger files (200 MB)
max_clipboard_size: 209715200

# Keep more backups (200 versions)
max_backup_count: 200

# Standard filename limit
max_filename_length: 200

# Use hidden backup directory
backup_dir_name: .pt-backups

# Deep recursive search (20 levels)
max_search_depth: 20
```

## Config Management Commands

### Create Config File

```bash
# Create pt.yml in current directory
pt config init

# Create config in specific location
pt config init ~/pt.yml
pt config init ~/.config/pt/pt.yml
```

### View Current Configuration

```bash
pt config show
```

Output example:
```
Current PT Configuration:

Max Clipboard Size: 104857600 bytes (100.0 MB)
Max Backup Count: 100
Max Filename Length: 200 characters
Backup Directory: backup/
Max Search Depth: 10 levels

Config loaded from: ./pt.yml
```

### Show Config File Location

```bash
pt config path
```

This shows which config file is being used, or suggests locations if none exists.

## Use Cases

### Case 1: Large File Support

For working with large code files or documentation:

```yaml
max_clipboard_size: 524288000  # 500 MB
max_backup_count: 50
backup_dir_name: large-file-backups
```

### Case 2: Minimal Disk Usage

For systems with limited disk space:

```yaml
max_clipboard_size: 10485760   # 10 MB
max_backup_count: 20
max_search_depth: 3
```

### Case 3: Extensive Version History

For critical files needing long history:

```yaml
max_backup_count: 1000
backup_dir_name: .history
```

### Case 4: Fast Performance

For quick operations with minimal searching:

```yaml
max_search_depth: 3
max_backup_count: 30
```

### Case 5: Developer Workflow

Optimized for code development:

```yaml
max_clipboard_size: 52428800   # 50 MB (code files)
max_backup_count: 100
backup_dir_name: .pt-versions
max_search_depth: 15           # Search through project structure
```

## Config Precedence

1. Config file (if found)
2. Built-in defaults

Within a config file, any omitted values use their defaults.

## Validation

PT validates all config values and falls back to defaults if:
- Value is out of range
- Value is invalid type
- Config file is malformed

Warnings are logged to stderr when validation fails.

## Tips

1. **Start with defaults**: Use `pt config init` to see all options
2. **Test changes**: Use `pt config show` to verify settings
3. **Version control**: Add `pt.yml` to your project's git repo for team consistency
4. **Per-project config**: Place `pt.yml` in project root for project-specific settings
5. **Global config**: Place in `~/.config/pt/` for system-wide defaults

## Troubleshooting

### Config not being loaded

Check file location:
```bash
pt config path
```

Ensure file is valid YAML:
```bash
# Test YAML syntax
python3 -c "import yaml; yaml.safe_load(open('pt.yml'))"
```

### Values not taking effect

View current config:
```bash
pt config show
```

Check stderr for validation warnings:
```bash
pt config show 2>&1 | grep Warning
```

### Reset to defaults

Simply rename or delete the config file:
```bash
mv pt.yml pt.yml.backup
pt config show  # Will show defaults
```

## Environment Variables

Currently, PT does not support environment variable configuration. Use config files for customization.

## Future Enhancements

Planned features for future versions:

- [ ] Environment variable overrides
- [ ] Command-line flag overrides
- [ ] Config validation command
- [ ] Config migration tool
- [ ] Multiple config profiles
- [ ] Config encryption support

## Examples Repository

See the `examples/configs/` directory for more configuration examples:

- `minimal.yml` - Minimal disk usage
- `developer.yml` - Developer workflow
- `large-files.yml` - Large file support
- `ci-cd.yml` - CI/CD pipeline usage

## Support

For config-related issues:
- Check [GitHub Issues](https://github.com/cumulus13/pt/issues)
- Email: cumulus13@gmail.com