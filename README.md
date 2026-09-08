# rpass

Read-only KeePass TUI. Browse a `.kdbx` vault from the terminal, fuzzy-search entries, and copy passwords to the clipboard.

## Requirements

- [Go](https://go.dev/dl/) 1.25 or newer
- A KeePass 2 database (`.kdbx`), password only (no keyfile)
- Linux clipboard: `xclip` or `xsel` (`sudo apt install xclip`)

## Install (Ubuntu)

From the repo root:

```bash
chmod +x install.sh
./install.sh
```

That builds `rpass` and copies it to `~/.local/bin/rpass` (no sudo). On current Ubuntu, `~/.local/bin` is already on your PATH after a new login. If `rpass` is not found, add this to `~/.bashrc` and open a new terminal:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

System-wide (needs write access to `/usr/local/bin`):

```bash
PREFIX=/usr/local ./install.sh
```

Uninstall:

```bash
rm -f ~/.local/bin/rpass
# or
sudo rm -f /usr/local/bin/rpass
```

### Build without installing

```bash
go build -o rpass .
./rpass ~/vault.kdbx
```

## Usage

```
rpass              Open the last used database (or pick from recents)
rpass <file.kdbx>  Open a database file
rpass --help
```

Relative paths work; recents are stored as absolute paths in `~/.config/rpass/recents.json`. Missing files are dropped from the list. If there is only one valid recent, the picker is skipped.

The first time, pass a file:

```bash
rpass ~/Documents/vault.kdbx
```

After that, `rpass` is enough.

## Keys

| Key | Action |
|-----|--------|
| `j` `k` / arrows | Move |
| `enter` `l` | Open folder or entry |
| `h` `←` | Parent folder |
| `/` `ctrl+f` | Search |
| `↓` `tab` | Jump from search box to results |
| `y` | Copy password (clipboard clears after 20s) |
| `u` | Copy username |
| `w` | Copy URL |
| `o` | Open that search hit's folder in the tree |
| `?` | Help |
| `esc` | Back (search → tree → recents → quit) |
| `q` | Quit (except while typing) |
| `ctrl+c` | Quit always |

Search is fuzzy on title, username, URL, folder path, and notes. The password field is not searched.

## Notes

- Read-only: it never writes the `.kdbx` file.
- Recents store paths only, never the master password.
- Built for a local terminal. Clipboard over SSH is not supported.
