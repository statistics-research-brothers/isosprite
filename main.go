package main

import (
	"fmt"
	"os"
)

const helpText = `isosprite - minimal ISO9660 image tool

Usage:
  isoforge create <source_folder> <output.iso>
  isoforge extract <input.iso> <output_folder>
  isoforge -h | --help

Commands:
  create    build an ISO9660 image from a folder
  extract   extract an ISO9660 image into a folder
`

func main() {
	args := os.Args[1:]
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		fmt.Print(helpText)
		return
	}
	switch args[0] {
	case "create":
		if len(args) != 3 {
			fmt.Println("usage: isosprite create <source_folder> <output.iso>")
			os.Exit(1)
		}
		if err := CreateISO(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	case "extract":
		if len(args) != 3 {
			fmt.Println("usage: isosprite extract <input.iso> <output_folder>")
			os.Exit(1)
		}
		if err := ExtractISO(args[1], args[2]); err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		}
	default:
		fmt.Print(helpText)
		os.Exit(1)
	}
}
