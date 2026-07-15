# isosprite

A minimal ISO9660 image tool. Builds a disc image from a folder, and
extracts a disc image back to a folder.

![Language](https://img.shields.io/badge/language-Go-00ADD8)
![License](https://img.shields.io/badge/license-Boost%201.0-blue)
![Platform](https://img.shields.io/badge/platform-linux%20%7C%20android%20%7C%20macos%20%7C%20windows-lightgrey)
![Dependencies](https://img.shields.io/badge/dependencies-zero-brightgreen)

## Overview

`isosprite` implements the ECMA-119 (ISO9660) format directly: Primary Volume
Descriptor, path tables, directory records, and file extents are constructed
by hand. There is no dependency on `mkisofs`, `genisoimage`, `xorriso`, or any
other external program, and no third-party Go module is imported anywhere in
the codebase.

## Language

Go (1.21+). Single binary, no runtime dependencies.

## Libraries Used

None. Only the Go standard library is used:

| Package | Purpose |
|---|---|
| `os` | file and directory I/O |
| `path/filepath` | path joining |
| `encoding/binary` | little/big-endian field encoding |
| `sort` | alphabetical ordering of directory entries |
| `strings`, `fmt`, `time` | name sanitizing, formatting, timestamps |

## Build

```
make build
```

Runs `CGO_ENABLED=0 go build -trimpath -ldflags="-s -w"`, producing a small,
statically linked binary with no debug symbols and no cgo.

## Usage

```
isosprite create <source_folder> <output.iso>
isosprite extract <input.iso> <output_folder>
isosprite -h | --help
```

## Is it safe?

- **No external processes.** The tool never shells out; it only reads and
  writes files under the paths you give it.
- **No network access.** Nothing is transmitted anywhere.
- **No cgo, no third-party code.** The entire attack surface is the Go
  standard library plus the code in this repository, which is small enough
  to read end to end.
- **Bounded, deterministic writes.** Output file size is computed up front
  from the source folder and written with `WriteAt` at fixed offsets, so
  there is no unbounded recursion or unchecked memory growth from malformed
  input during creation. Extraction trusts the on-disk directory records byte
  for byte, so only run `extract` against ISO images you trust — as with any
  archive tool, a maliciously crafted image could point extents outside
  their intended bounds.

## Is it scientific or a fast script?

It's a fast, single-purpose utility script, not a scientific/numerical tool.
There's no floating-point computation, no benchmarked algorithm, and no
research claim to validate. Performance is dominated by disk I/O
(`os.ReadFile` / `WriteAt`), and the implementation favors a small, readable
binary over throughput tuning — there's no buffering, streaming, or
parallelism, so very large source trees are read and written in a single
pass per file.

## Limitations

Plain ISO9660 (ECMA-119) only — no Rock Ridge, no Joliet. Filenames are
uppercased, restricted to `A-Z 0-9 _ .`, and truncated to fit ISO9660 level-2
identifier limits (`;1` version suffix on files). This is the trade-off for
keeping the implementation dependency-free and small: original filename
casing and long names are not preserved on extraction.

## License

Boost Software License 1.0. See [`LICENSE`](./LICENSE).
