package lsjson

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"github.com/rclone/rclone/cmd"
	"github.com/rclone/rclone/cmd/ls/lshelp"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/flags"
	"github.com/rclone/rclone/fs/operations"
	"github.com/spf13/cobra"
)

var (
	opt      operations.ListJSONOpt
	statOnly bool
)

func init() {
	cmd.Root.AddCommand(commandDefinition)
	cmdFlags := commandDefinition.Flags()
	flags.BoolVarP(cmdFlags, &opt.Recurse, "recursive", "R", false, "Recurse into the listing.")
	flags.BoolVarP(cmdFlags, &opt.ShowHash, "hash", "", false, "Include hashes in the output (may take longer).")
	flags.BoolVarP(cmdFlags, &opt.NoModTime, "no-modtime", "", false, "Don't read the modification time (can speed things up).")
	flags.BoolVarP(cmdFlags, &opt.NoMimeType, "no-mimetype", "", false, "Don't read the mime type (can speed things up).")
	flags.BoolVarP(cmdFlags, &opt.ShowEncrypted, "encrypted", "M", false, "Show the encrypted names.")
	flags.BoolVarP(cmdFlags, &opt.ShowOrigIDs, "original", "", false, "Show the ID of the underlying Object.")
	flags.BoolVarP(cmdFlags, &opt.FilesOnly, "files-only", "", false, "Show only files in the listing.")
	flags.BoolVarP(cmdFlags, &opt.DirsOnly, "dirs-only", "", false, "Show only directories in the listing.")
	flags.StringArrayVarP(cmdFlags, &opt.HashTypes, "hash-type", "", nil, "Show only this hash type (may be repeated).")
	flags.BoolVarP(cmdFlags, &statOnly, "stat", "", false, "Just return the info for the pointed to file.")
}

var commandDefinition = &cobra.Command{
	Use:   "lsjson remote:path",
	Short: `List directories and objects in the path in JSON format.`,
	Long: `List directories and objects in the path in JSON format.

The output is an array of Items, where each Item looks like this

   {
      "Hashes" : {
         "SHA-1" : "f572d396fae9206628714fb2ce00f72e94f2258f",
         "MD5" : "b1946ac92492d2347c6235b4d2611184",
         "DropboxHash" : "ecb65bb98f9d905b70458986c39fcbad7715e5f2fcc3b1f07767d7c83e2438cc"
      },
      "ID": "y2djkhiujf83u33",
      "OrigID": "UYOJVTUW00Q1RzTDA",
      "IsBucket" : false,
      "IsDir" : false,
      "MimeType" : "application/octet-stream",
      "ModTime" : "2017-05-31T16:15:57.034468261+01:00",
      "Name" : "file.txt",
      "Encrypted" : "v0qpsdq8anpci8n929v3uu9338",
      "EncryptedPath" : "kja9098349023498/v0qpsdq8anpci8n929v3uu9338",
      "Path" : "full/path/goes/here/file.txt",
      "Size" : 6,
      "Tier" : "hot",
   }

If --hash is not specified the Hashes property won't be emitted. The
types of hash can be specified with the --hash-type parameter (which
may be repeated). If --hash-type is set then it implies --hash.

If --no-modtime is specified then ModTime will be blank. This can
speed things up on remotes where reading the ModTime takes an extra
request (e.g. s3, swift).

If --no-mimetype is specified then MimeType will be blank. This can
speed things up on remotes where reading the MimeType takes an extra
request (e.g. s3, swift).

If --encrypted is not specified the Encrypted won't be emitted.

If --dirs-only is not specified files in addition to directories are
returned

If --files-only is not specified directories in addition to the files
will be returned.

if --stat is set then a single JSON blob will be returned about the
item pointed to. This will return an error if the item isn't found.
However on bucket based backends (like s3, gcs, b2, azureblob etc) if
the item isn't found it will return an empty directory as it isn't
possible to tell empty directories from missing directories there.

The Path field will only show folders below the remote path being listed.
If "remote:path" contains the file "subfolder/file.txt", the Path for "file.txt"
will be "subfolder/file.txt", not "remote:path/subfolder/file.txt".
When used without --recursive the Path will always be the same as Name.

If the directory is a bucket in a bucket based backend, then
"IsBucket" will be set to true. This key won't be present unless it is
"true".

The time is in RFC3339 format with up to nanosecond precision.  The
number of decimal digits in the seconds will depend on the precision
that the remote can hold the times, so if times are accurate to the
nearest millisecond (e.g. Google Drive) then 3 digits will always be
shown ("2017-05-31T16:15:57.034+01:00") whereas if the times are
accurate to the nearest second (Dropbox, Box, WebDav, etc.) no digits
will be shown ("2017-05-31T16:15:57+01:00").

The whole output can be processed as a JSON blob, or alternatively it
can be processed line by line as each item is written one to a line.
` + lshelp.Help,
	RunE: func(command *cobra.Command, args []string) error {
		cmd.CheckArgs(1, 1, command, args)
		var fsrc fs.Fs
		var remote string
		if statOnly {
			fsrc, remote = cmd.NewFsFile(args[0])
		} else {
			fsrc = cmd.NewFsSrc(args)
		}
		cmd.Run(false, false, command, func() error {
			if statOnly {
				item, err := operations.StatJSON(context.Background(), fsrc, remote, &opt)
				if err != nil {
					return err
				}
				out, err := json.MarshalIndent(item, "", "\t")
				if err != nil {
					return errors.Wrap(err, "failed to marshal list object")
				}
				_, err = os.Stdout.Write(out)
				if err != nil {
					return errors.Wrap(err, "failed to write to output")
				}
				fmt.Println()
			} else {
				fmt.Println("[")
				first := true
				err := operations.ListJSON(context.Background(), fsrc, remote, &opt, func(item *operations.ListJSONItem) error {
					out, err := json.Marshal(item)
					if err != nil {
						return errors.Wrap(err, "failed to marshal list object")
					}
					if first {
						first = false
					} else {
						fmt.Print(",\n")
					}
					_, err = os.Stdout.Write(out)
					if err != nil {
						return errors.Wrap(err, "failed to write to output")
					}
					return nil
				})
				if err != nil {
					return err
				}
				if !first {
					fmt.Println()
				}
				fmt.Println("]")
			}
			return nil
		})
		return nil
	},
}
