# cutechan [![Build Status](https://travis-ci.org/cutechan/cutechan.svg?branch=master)](https://travis-ci.org/cutechan/cutechan)

Platforms: Linux, OSX, Win64

License: GNU AGPL

## Features

<details><summary>Posts and posting</summary>

- Character by character post updates
- Hovering quick reply for post authoring
- Dice roll, coin flip and eightball commands
- Desktop notifications  and "(You)" links on quote
- Post link hover previews, including cross-thread
- Inline post link expansion
- Optional relative post timestamps
- Non-temporal and recursive post linking
- No posts per thread or threads per board limit
- Forced anonymity display mode
- Post hiding
- Option to display only the last 100 posts in a thread
- Optional automatic deletion of unused threads and boards
- Automatic URL linkification
- Automatic intelligent quoting of selected text, when quoting a post
- Live programming code tags with syntax highlighting
- Automatic open post recovery after a disconnect
- Toggleable non-live post creation
- Keyboard post navigation
- Explicitly visible sage
- Responsive seen post detection

</details>

<details><summary>Files and images</summary>

- JPEG, PNG, APNG, WEBM, MP3, MP4, OGG, PDF, ZIP, 7Z, TAR.GZ and TAR.XZ are supported
- Transparent PNG and GIF thumbnails
- Configurable size limits
- Inbuilt reverse image search
- No file is ever thumbnailed or stored twice, reducing server load and disk space usage
- Any file already present on the server is "uploaded and thumbnailed" instantly
- Title metadata extraction
- Gallery mode

</details>

<details><summary>Performance</summary>

- Low memory and CPU usage
- No frameworks and optimized code on both client and server
- File upload processing written in C with GraphicsMagick and ffmpeg
- Inbuilt custom multi-level LRU cache

</details>

<details><summary>Client UI</summary>

- Works with all modern and most outdated browsers (such as PaleMoon)
- Works with JavaScript disabled browsers
- Multiple themes
- Custom user-set backgrounds and CSS
- Mascots
- Configurable keyboard shortcuts
- Work mode aka Boss key
- Customisable top banner board link list
- Optional animated GIF thumbnails
- Settings export/import to/from JSON file

</details>

<details><summary>Board administration/moderation</summary>

- Support for both centralized and 8chan-style board ownership
- Global admin -> users notification system
- User board creation and configuration panels
- 4 tier staff system
- Board-level and global bans
- Transparent post deletion
- Viewing of all post made by same IP
- Option to disable search indexing on board
- Sticky threads
- Public ban list
- Public moderation log

</details>

<details><summary>Internationalization</summary>

- Client almost entirely localized in multiple languages
- More languages can be added by editing simple JSON files

</details>

<details><summary>Miscellaneous</summary>

- Documented public JSON API
- Optional R/a/dio Now Playing banner
- Synchronized time counters (for group watching sessions and such)
- Thread-level connected unique IP counter
- Internal captcha system

</details>

## Runtime dependencies
* [PostgresSQL](https://www.postgresql.org/download/) >= 9.5
* ffmpeg >= 3.0 shared libraries (libavcodec, libavutil, libavformat) compiled with:
    * libvpx
    * libvorbis
    * libopus
    * libtheora
    * libx264
    * libmp3lame
* GraphicsMagick shared library compiled with:
    * zlib
    * libpng
    * libjpeg
    * postscript

## Building from source
A reference list of commands can be found in `./docs/installation.md`

### Build dependencies
* [Go](https://golang.org/doc/install) >= 1.9 (for building server)
* [Node.js](https://nodejs.org) >= 5.0 (for building client)
* GCC or Clang
* git
* make
* pthread
* pkg-config
* ffmpeg and GraphicsMagick development files

### Linux and OSX
* Run `make`

### Windows
* Install [MSYS2](https://sourceforge.net/projects/msys2/)
* Open MSYS2 shell
* Install dependencies listed above with the `mingw-w64-x86_64-` prefix with
pacman
* Navigate to the cutechan root directory
* Run `make`

## Setup
* See `./cutechan help` for server operation
* Login into the "admin" account via the infinity symbol in the top banner with
the password "password"
* Change the default password
* Create a board from the administration panel
* Configure server from the administration panel

## Development
* See `./docs` for more documentation
* `./cutechan` or `./cutechan debug` run the server in development mode
* `make server` and `make client` build the server and client separately
* `make watch` watches the file system for changes and incrementally rebuilds
the client
* `make update-deps` updates all dependencies
* `make clean` removes files from the previous compilation
* `make distclean` in addition to the above removes uploaded files and their
thumbnails
* To enable using Go tools in the project add the absolute path of `./go` to
your `$GOPATH` environment variable
