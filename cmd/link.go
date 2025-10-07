package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/jfrog/jfrog-cli-vm/cmd/utils"
	"github.com/urfave/cli/v2"
)

var Link = &cli.Command{
	Name:  "link",
	Usage: "Link a local jf binary into jfcm",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "from", Usage: "Path to the local jf binary", Required: true},
		&cli.StringFlag{Name: "name", Usage: "Version name to assign", Required: true},
	},
	Action: func(c *cli.Context) error {
		from := c.String("from")
		name := c.String("name")

		if _, err := os.Stat(from); os.IsNotExist(err) {
			return fmt.Errorf("no such file: %s", from)
		}

		targetDir := filepath.Join(utils.jfcmVersions, name)
		targetBin := filepath.Join(targetDir, utils.BinaryName)
		err := os.MkdirAll(targetDir, 0755)
		if err != nil {
			return err
		}

		src, err := os.Open(from)
		if err != nil {
			return err
		}
		defer func(src *os.File) {
			_ = src.Close()
		}(src)

		dst, err := os.Create(targetBin)
		if err != nil {
			return err
		}
		defer func(dst *os.File) {
			_ = dst.Close()
		}(dst)

		if _, err := io.Copy(dst, src); err != nil {
			return err
		}
		if err := os.Chmod(targetBin, 0755); err != nil {
			return err
		}

		fmt.Printf("✅ Linked %s as jfcm version %s\n", from, name)
		return nil
	},
}
