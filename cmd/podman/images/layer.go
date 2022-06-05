package images

import (
	"context"
	"os"
	"strings"

	"github.com/containers/common/pkg/completion"
	"github.com/containers/common/pkg/util"
	"github.com/containers/podman/v4/cmd/podman/common"
	"github.com/containers/podman/v4/cmd/podman/parse"
	"github.com/containers/podman/v4/cmd/podman/registry"
	"github.com/containers/podman/v4/libpod/define"
	"github.com/containers/podman/v4/pkg/domain/entities"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	layerDescription = `manipulate image layers.`

	layerCommand = &cobra.Command{
		Use:   "layer [options] IMAGE [IMAGE...]",
		Short: "Layer mainpulator",
		Long:  layerDescription,
		RunE:  layer,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.Errorf("need at least 1 argument")
			}
			format, err := cmd.Flags().GetString("format")
			if err != nil {
				return err
			}
			if !util.StringInSlice(format, common.ValidSaveFormats) {
				return errors.Errorf("format value must be one of %s", strings.Join(common.ValidSaveFormats, " "))
			}
			return nil
		},
		ValidArgsFunction: common.AutocompleteImages,
		Example: `podman modify --quiet -o myimage.tar imageID
  podman modify --format docker-dir -o ubuntu-dir ubuntu
  podman modify > alpine-all.tar alpine:latest`,
	}

	imageModifyCommand = &cobra.Command{
		Args:              layerCommand.Args,
		Use:               layerCommand.Use,
		Short:             layerCommand.Short,
		Long:              layerCommand.Long,
		RunE:              layerCommand.RunE,
		ValidArgsFunction: layerCommand.ValidArgsFunction,
		Example: `podman image modify --quiet -o myimage.tar imageID
  podman image modify --format docker-dir -o ubuntu-dir ubuntu
  podman image modify > alpine-all.tar alpine:latest`,
	}
)

var (
	layerOpts entities.ImageSaveOptions
)

func init() {
	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: layerCommand,
	})
	layerFlags(layerCommand)

	registry.Commands = append(registry.Commands, registry.CliCommand{
		Command: imageModifyCommand,
		Parent:  imageCmd,
	})
	layerFlags(imageModifyCommand)
}

func layerFlags(cmd *cobra.Command) {
	flags := cmd.Flags()

	flags.BoolVar(&layerOpts.Compress, "compress", false, "Compress tarball image layers when saving to a directory using the 'dir' transport. (default is same compression type as source)")

	flags.BoolVar(&layerOpts.OciAcceptUncompressedLayers, "uncompressed", false, "Accept uncompressed layers when copying OCI images")

	formatFlagName := "format"
	flags.StringVar(&layerOpts.Format, formatFlagName, define.V2s2Archive, "Modify image to oci-archive, oci-dir (directory with oci manifest type), docker-archive, docker-dir (directory with v2s2 manifest type)")
	_ = cmd.RegisterFlagCompletionFunc(formatFlagName, common.AutocompleteImageSaveFormat)

	outputFlagName := "output"
	flags.StringVarP(&layerOpts.Output, outputFlagName, "o", "", "Write to a specified file (default: stdout, which must be redirected)")
	_ = cmd.RegisterFlagCompletionFunc(outputFlagName, completion.AutocompleteDefault)

	flags.BoolVarP(&layerOpts.Quiet, "quiet", "q", false, "Suppress the output")
	flags.BoolVarP(&layerOpts.MultiImageArchive, "multi-image-archive", "m", containerConfig.Engine.MultiImageArchive, "Interpret additional arguments as images not tags and create a multi-image-archive (only for docker-archive)")
}

func replace(oldLayer string, newLayer string) error {
	return nil
}

func layer(cmd *cobra.Command, args []string) (finalErr error) {
	var (
		tags      []string
		succeeded = false
	)
	if cmd.Flag("compress").Changed && (layerOpts.Format != define.OCIManifestDir && layerOpts.Format != define.V2s2ManifestDir) {
		return errors.Errorf("--compress can only be set when --format is either 'oci-dir' or 'docker-dir'")
	}
	if len(layerOpts.Output) == 0 {
		layerOpts.Quiet = true
		fi := os.Stdout
		if term.IsTerminal(int(fi.Fd())) {
			return errors.Errorf("refusing to modify to terminal. Use -o flag or redirect")
		}
		pipePath, cleanup, err := setupPipe()
		if err != nil {
			return err
		}
		if cleanup != nil {
			defer func() {
				errc := cleanup()
				if succeeded {
					writeErr := <-errc
					if writeErr != nil && finalErr == nil {
						finalErr = writeErr
					}
				}
			}()
		}
		layerOpts.Output = pipePath
	}
	if err := parse.ValidateFileName(layerOpts.Output); err != nil {
		return err
	}
	if len(args) > 1 {
		tags = args[1:]
	}

	oldLayer := "old"
	newLayer := "new"
	err := replace(oldLayer, newLayer)
	if err == nil {
		succeeded = true
	}
	return err
}
