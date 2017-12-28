// +build mage

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/magefile/mage/mg" // mg contains helpful utility functions, like Deps
	"github.com/magefile/mage/sh"
)

const (
	BasePackage = "github.com/Scalingo/sand"
)

// Default target to run when none is specified
// If not set, running mage will list available targets
var Default = Build

// A build step that requires additional params, or platform specific steps for example
func Build(ctx context.Context) error {
	mg.CtxDeps(ctx, InstallDeps)
	fmt.Println("Building…")
	cmd := exec.Command("go", "build", "-i", "github.com/Scalingo/sand/cmd/sand-agent")
	return cmd.Run()
}

// Run specs on the project
func Test(ctx context.Context) error {
	fmt.Println("Testing…")
	return sh.RunV("go", "test", "./...")
}

// Generate the mocks of used interface for `mockgen/gomock`
func GenerateMocks(ctx context.Context) error {
	mg.CtxDeps(ctx, InstallTestDeps)

	mocks := []struct {
		Package   string
		Interface string
	}{
		{Package: "netlink", Interface: "Handler"},
		{Package: "ipallocator", Interface: "IPAllocator"},
		{Package: "netnsbuilder", Interface: "Manager"},
		{Package: "idmanager", Interface: "Manager"},
		{Package: "store", Interface: "Store"},
		{Package: "endpoint", Interface: "Repository"},
		{Package: "network", Interface: "Repository"},
		{Package: "network/netmanager", Interface: "NetManager"},
		{Package: "network/overlay", Interface: "NetworkEndpointListener"},
	}

	for _, mock := range mocks {
		fmt.Printf("Building mock of %v#%v\n", mock.Package, mock.Interface)

		mockPackage := mock.Package + "mock"
		mockDirectory := fmt.Sprintf("test/mocks/%s", mockPackage)
		filePackage := filepath.Base(mockPackage)
		mockFile := fmt.Sprintf("%s/%s_mock.go", mockDirectory, strings.ToLower(mock.Interface))

		err := os.MkdirAll(mockDirectory, 0755)
		if err != nil {
			return err
		}
		err = sh.Run(
			"mockgen",
			"-destination", mockFile,
			"-package", filePackage,
			fmt.Sprintf("%s/%s", BasePackage, mock.Package),
			mock.Interface,
		)
		if err != nil {
			return err
		}

		err = sh.Run("sed", "-i", fmt.Sprintf("s,%s/vendor/,,", BasePackage), mockFile)
		if err != nil {
			return err
		}

		err = sh.RunV("goimports", "-w", mockFile)
		if err != nil {
			return err
		}
	}
	return nil
}

// A custom install step if you need your bin someplace other than go/bin
func Install(ctx context.Context) error {
	mg.CtxDeps(ctx, Build)
	fmt.Println("Installing…")
	return sh.Run("go", "install", "github.com/Scalingo/sand/cmd/sand-agent")
}

// Manage your deps, or running package managers.
func InstallDeps() error {
	return nil
}

// Install test dependencies like gomock/mockgen
func InstallTestDeps() error {
	cmd := exec.Command("go", "get", "github.com/golang/mock/gomock")
	err := cmd.Run()
	if err != nil {
		return err
	}
	cmd = exec.Command("go", "get", "github.com/golang/mock/mockgen")
	return cmd.Run()
}

// Clean up after yourself
func Clean() {
	fmt.Println("Cleaning…")
	os.RemoveAll("./sand-agent")
}
