# trebup

The installer for Treb - update or revert to a specific Treb version with ease.

`trebup` supports installing and managing multiple versions.

## Installing

```sh
curl -L https://raw.githubusercontent.com/trebuchet-org/treb-cli/main/trebup/install | bash
```

This will install `trebup` and add `~/.treb/bin` to your PATH.

## Usage

To install the latest **stable** version:

```sh
trebup
```

To **install** a specific **version**:

```sh
trebup --install v0.2.0
```

To install the **nightly** version:

```sh
trebup --install nightly
```

To **list** all installed versions:

```sh
trebup --list
```

To **switch** between installed versions:

```sh
trebup --use v0.1.0
```

To install from a specific **branch**:

```sh
trebup --branch feature/new-feature
```

To install from a **fork's main branch**:

```sh
trebup --repo someuser/treb-cli
```

To install a **specific branch in a fork**:

```sh
trebup --repo someuser/treb-cli --branch feature-branch
```

To install from a **specific Pull Request**:

```sh
trebup --pr 123
```

To install from a **specific commit**:

```sh
trebup --commit 94bfdb2
```

To install from a **local repository** (for development):

##### Note: --branch, --repo, and --version flags are ignored during local installations.

```sh
trebup --path ~/git/treb-cli
```

To **update trebup** itself to the latest version:

```sh
trebup --update
```

---

**Tip**: All flags have single character shortcuts! Use `-i` instead of `--install`, `-b` instead of `--branch`, etc.

---

## How it works

`trebup` manages Treb installations in `~/.treb/`:
- Downloads pre-built binaries from GitHub releases (fast, no build dependencies)
- Supports building from source for development branches/PRs (requires Go)
- Allows multiple versions to be installed side-by-side
- Easy switching between versions with `--use`

## Directory structure

```
~/.treb/
├── bin/            # Active treb binary and trebup
├── versions/       # All installed versions
│   ├── stable/     # Latest stable release
│   ├── nightly/    # Latest nightly build
│   └── v0.2.0/     # Specific version
└── cache/          # Git cache for source builds
```

## Uninstalling

To uninstall trebup and all Treb installations:

```sh
rm -rf ~/.treb
```

Then remove the PATH entry from your shell configuration file (`.bashrc`, `.zshenv`, etc.).