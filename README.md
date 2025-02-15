# ROMCopyEngine

*Your ROM collection is perfect and meticulously organized. But this device/OS wants boxart in a folder called `Imgs`. One OS wants a `gameslist.xml` file, but another one wants `miyoogamelist.xml`. Another one calls `nes` by `fc` and `snes` by `supernes`. How can you get your ROMs loaded, and consistently updated, on this device with minimal headache?*

## Use ROMCopyEngine.

![Alt text](/screenshot.png?raw=true "ROMCopyEngine in action")


# Installation

## via Releases

Check out the newest release from [the releases page](https://github.com/jkingsman/ROMCopyEngine/releases) or glance at the right sidebar.

Download the appropriate binary for your platform, unzip it, and use it! If you're on Linux, I've got binaries as well as a deb and an RPM.

## via `go get`

If you've got go, or [want to install it](https://go.dev/doc/install):

```
go install github.com/jkingsman/ROMCopyEngine@latest
romcopyengine --help
```

## from source

```
git clone https://github.com/jkingsman/ROMCopyEngine.git
cd ROMCopyEngine
go run romcopyengine.go
```

## Example usages

### Example 1

```bash
./romcopyengine \
    --sourceDir /mnt/d/ROMs/ \
    --targetDir /mnt/i/ \
    --mapping psx:PS1 \
    --copyExclude "*.chd"
```

Copy PS1 games from your `psx` directory to the device's `PS1` directory, but exclude `.chd` files

**Note that globs are path-local to the platform folder! If you want to work on ALL `*chd` files regardless of hierarchy, you need to doublestar glob (e.g. `**/*.chd`)**

### Example 2

```bash
romcopyengine \
    --sourceDir /mnt/d/ROMs/ \
    --targetDir /mnt/i/ \
    --mapping psx:PS1 \
    --explodeDir multidisk \
    --rewrite "*.m3u:./multidisk:./"
```

Copy PS1 games, and for games in the folder called `multidisk`, move them to the same folder level as other games and update any `.m3u` files to reflect that.

### Example 3

```bash
romcopyengine \
    --sourceDir /mnt/d/ROMs/ \
    --targetDir /mnt/i/ \
    --mapping psx:PS1 \
    --mapping snes:SFC \
    --mapping nes:FC \
    --mapping gg:GAMEGEAR \
    --copyExclude '**/*.old' \
    --copyExclude '**/*.backup' \
    --explodeDir multidisk \
    --explodeDir images \
    --rename gameslist.xml:data.xml \
    --rename all_games.txt:_all_games.txt \
    --rewritesAreRegex \
    --rewrite "*.m3u:\./multidisk:./" \
    --rewrite "*.xml:\.\./.*?/images:./"
```

An incredibly full featured invocation:

* Copies from `/mnt/d/ROMs/` to `/mnt/i/`, copying the contents of:
    * `/mnt/d/ROMs/psx` to `/mnt/i/PS1`
    * `/mnt/d/ROMs/snes` to `/mnt/i/SFC`
    * `/mnt/d/ROMs/nes` to `/mnt/i/FC`
    * `/mnt/d/ROMs/gg` to `/mnt/i/GAMEGEAR`
* Exclude any files ending in `.old` or `.backup`
* Copy the contents of the folders `multidisk` and `images` in each of the console folders to the top of the console folder (e.g. `/mnt/i/PS1/images/file.jpg` moves to `/mnt/i/PS1/file.jpg`) and delete the now-empty folders
* Rename `gameslist.xml` to `data.xml` and `all_games.txt` to `_all_games.txt`
* Flag rewrite rules as regular expressions
* Replace all instances of `./multidisk` with `./` in `.m3u` files (because disks moved during explosion)
* Replace all instances of `../<anything>/images` (e.g. `../snes/images` or `../gg/images`) to `./` in `.xml` files (because images moved during explosion, and assuming something like Skraper has done directories a bit weird as it can tend to do)

## Command line option overview

*This is basically availble via romcopyengine --help.*

### Source, destination, and their relationship

* `--sourceDir <path>`: Required. The source directory containing platform folders (`snes`, `gba`, etc.) to be copied from e.g. `C:\ROMS` or `/home/ROMS`.

* `--targetDir <path>`: Required. Target directory (usually on device) containing platform folders (`snes`, `gba`, etc.), e.g. `J:\` or `/media/usb-drive/`.

* `--mapping <source:destination>`: At least one required. A mapping of source platform folder to destination platform folder for the ROMs in the format `source:destination`. For example, `--mapping snes:SFC --mapping gg:GameGear` would copy the contents of the `sourceDir`'s `snes` folder to the `targetDir`'s `SFC` folder and the contents of the sourceDir's `gg` folder to the targetDir's 'GameGear' folder.

### Choosing what to copy

* `--copyInclude <glob>`: Copy only files and folders within each mapping which match the given glob. For example, `--copyInclude '*_favorite*'` would only copy files/folders containing `_favorite`; `--copyInclude '*.xml'` would only copy XML files. Remember to single quote your glob to prevent shell expansion. Multiples of this flag are allowed (OR relation). Supports globstar (e.g. `**/*.png`).

* `--copyExclude <glob>`: Copy only files and folders within each mapping which do NOT match the given glob. For example, `--copyExclude '*.xml'` would copy all files except those ending in `.xml`. Remember to single quote your glob. Multiples of this flag are allowed (AND relation). Processed after --copyInclude entries.

### Mutating file names, locations, and contents

* `--explodeDir <dirname>`: Provides a directory name contained in a ROM folder that should have its contents copied to the parent directory for that system, then delete the empty folder. For example, `--explodeDir images` would copy the contents of the image directory into its parent folder. Commonly used to bring boxart images out of an `images` directory. Multiples allowed.

* `--rename <old:new>`: Rename files or folders from a given name to a given name after copy. For example, `--rename gameslist.xml:miyoogameslist.xml` would rename all occurrences of `gameslist.xml` in all folders to `miyoogameslist.xml`; `--rename images:Imgs` could be used to rename image folders. Multiples of this flag are allowed.

* `--rewrite <glob>:<search>:<replace>`: For a given file glob, execute a find and replace on all matching files. Useful for fixing paths in XML files. Remember to single quote globs to prevent shell expansion. For example, `--rewrite "*.xml:\.\./.*?/images:./images"` would replace `../images` with `./images` in all XML files. Multiples allowed.

* `--rewritesAreRegex`: Optional. When set, the search term in any --rewrite flag is interpreted as a Golang regular expression.


### Operations

* `--cleanTarget`: Optional. Delete all files in the destination platform folder before copying ROMs in.

* `--skipConfirm`: Optional. Skip all confirmations and execute the copy process.

* `--dryRun`: Optional. Don't execute any file copies or operations; just print what would be done.

## Warnings

ROMCopyEngine will always overwrite destination files without prompting. Use `--dryRun` if you're not sure whether something would get copied.

File rename (`--rename`) and rewrite (`--rewrite`) operate on ALL files in the destination platform folder. If there are already files there and you don't choose to `--cleanTarget` to remove them, renames and rewrites will run on them as well.

Note that globs are path-local to the platform folder! If you want to work on all types of a file everywhere, you need to doublestar glob (e.g. `**/*.png`).

The logical flow of copying is:

* Print configuration summary
* Display a warning if `--cleanTarget` is selected, confirmation hasn't been skipped (`--skipConfirm`), and this isn't a dry run (`--dryRun`)
* Display a continuation prompt if confirmation hasn't been skipped (`--skipConfirm`) and this isn't a dry run (`--dryRun`)
* For each directory mapping/platform:
    * Clean the destination directory/platform, if `--cleanTarget` is set, empty the directory
    * Copy files over according to `--copyInclude` or `--copyExclude` if included
    * Explode each directory listed for explosion (`--explodeDir`)
    * Process each rename specified (`--rename`)
    * Process each specified rewrite/find and replace (`--rewrite`)

The tests here are absolute GARBAGE. Terrible composition, and I didn't write most of my functions to BE super testable so things are coupled together in really odd ways. LLMs wrote basically the entire test suite, which is a terrible thing but a whole lot more than I usually have in terms of side project tests, so if it keeps me from breaking something obvious, sure, I'll take it. Apologies if you're trying to extend them though.

## Contributing

Please format your code and ensure tests pass.

Before PRing, run `gofmt -w **/*.go`. Test changes with `go test -v ./... && python3 testing/test_blackbox.py`.

I will release builds (please don't PR artifacts), but you can test your artifacts by running `./build.sh` without a suffix (otherwise it tries to push that tag).
