// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
)

// Default target to run when none is specified
// If not set, running mage will list available targets
// var Default = Build

// A build step that requires additional params, or platform specific steps for example
func Build(ctx context.Context) error {
	mg.CtxDeps(ctx, InstallDeps)
	fmt.Println("Building...")
	cmd := exec.Command("go", "build", "-i", "github.com/Scalingo/sand/cmd/sand-agent")
	return cmd.Run()
}

// A custom install step if you need your bin someplace other than go/bin
func Install(ctx context.Context) error {
	mg.CtxDeps(ctx, Build)
	fmt.Println("Installing...")
	cmd := exec.Command("go", "install", "github.com/Scaligno/sand/cmd/sand-agent")
	return cmd.Run()
}

// Manage your deps, or running package managers.
func InstallDeps() error {
	return nil
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning...")
	os.RemoveAll("./sand-agent")
}
