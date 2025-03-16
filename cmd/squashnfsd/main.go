package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"time"

	squashfs "github.com/afbjorklund/go-squashnfs/pkg/squashfs"
	"github.com/go-git/go-billy/v5"
	nfs "github.com/willscott/go-nfs"
	nfshelper "github.com/willscott/go-nfs/helpers"

	"github.com/spf13/cobra"
)

type ROFS struct {
	billy.Filesystem
}

// Capabilities exports the filesystem as readonly
func (ROFS) Capabilities() billy.Capability {
	return billy.ReadCapability | billy.SeekCapability
}

var rootCmd = &cobra.Command{
	Use:  "squashnfsd archive mountpoint",
	Args: cobra.ExactArgs(2),
	RunE: run,
}

func init() {
	rootCmd.PersistentFlags().IntVarP(&port, "port", "p", 0, "port")
	rootCmd.PersistentFlags().Int64VarP(&offset, "offset", "o", 0, "offset")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "quiet")
	rootCmd.PersistentFlags().BoolVarP(&unmount, "unmount", "u", false, "unmount and exit")
	rootCmd.PersistentFlags().BoolVarP(&root, "root", "r", false, "mount as root always")
}

var port int
var offset int64
var quiet bool
var unmount bool
var root bool

func mount(addr string, path string) error {
	host, p, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	share := host + ":" + "/mount"
	port, err := strconv.Atoi(p)
	if err != nil {
		return err
	}

	// allow server some time to start
	time.Sleep(500 * time.Millisecond)

	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("fuse-nfs"); err == nil && !root {
			vers := "version"
			url := fmt.Sprintf("nfs://%s?nfsport=%d&mountport=%d&%s=3", share, port, port, vers)
			cmd = exec.Command("fuse-nfs", "-n", url, "-m", path)
		} else {
			vers := "nfsvers"
			opt := fmt.Sprintf("port=%d,mountport=%d,%s=3,noacl,tcp", port, port, vers)
			cmd = exec.Command("sudo", "mount", "-o", opt, "-t", "nfs", share, path)
		}
	default:
		opt := fmt.Sprintf("port=%d,mountport=%d", port, port)
		cmd = exec.Command("mount", "-o", opt, "-t", "nfs", share, path)
	}
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func umount(path string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "linux":
		if _, err := exec.LookPath("fuse-nfs"); err == nil && !root {
			cmd = exec.Command("fusermount", "-u", path)
		} else {
			cmd = exec.Command("sudo", "umount", "-l", path)
		}
	default:
		cmd = exec.Command("umount", path)
	}
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func run(cmd *cobra.Command, args []string) error {
	if unmount {
		return umount(args[1])
	}

	f, err := os.Open(args[0])
	if err != nil {
		return err
	}
	bfs := squashfs.New(f, offset)

	listener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		return err
	}
	addr := listener.Addr().String()
	if !quiet {
		fmt.Printf("Server running at %s\n", addr)
	}

	go func() {
		err := mount(addr, args[1])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Mount failed: %v\n", err)
		}
	}()

	handler := nfshelper.NewNullAuthHandler(ROFS{bfs})
	cacheHelper := nfshelper.NewCachingHandler(handler, 1024)
	return nfs.Serve(listener, cacheHelper)
}

func main() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
