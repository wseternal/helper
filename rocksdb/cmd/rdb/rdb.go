package main

import (
	"flag"
	"fmt"
	"helper"
	"helper/iohelper/sink"
	"helper/logger"
	"helper/rocksdb"
	"io"
	"net/http"
	"os"
	"os/user"
	"time"
)

type Option struct {
	// global options
	dbpath   string
	output   string
	user     string
	verbose  bool
	readonly bool

	// subcommand common options
	rocksdb.RangeOption
	subCmdHelp bool

	// restore subcommand option
	restorePath string
	backupPath  string

	// http subcommand option
	addr string

	// purge subcommand option
	keepCnt uint
}

type Runtime struct {
	rdb *rocksdb.RDB
	snk *sink.Sink

	fsSubcommand *flag.FlagSet
	subCommand   string
}

type DBCmd struct {
	Option
	Runtime
}

var (
	cmd DBCmd
)

func init() {
	var userName string
	if user, err := user.Current(); err != nil {
		logger.LogW("get current user failed, %s\n", err)
		userName = "www"
	} else {
		userName = user.Username
	}
	flag.Usage = cmdUsage
	flag.StringVar(&cmd.dbpath, "d", "", "full path to rocksdb")
	flag.StringVar(&cmd.output, "o", "", "output file")
	flag.StringVar(&cmd.user, "u", userName, "running with specified username")
	flag.BoolVar(&cmd.verbose, "v", false, "enable verbose output")
	flag.BoolVar(&cmd.readonly, "r", false, "open target db in readonly mode")

	cmd.fsSubcommand = flag.NewFlagSet("", flag.ExitOnError)
	cmd.fsSubcommand.BoolVar(&cmd.subCmdHelp, "h", false, "show this usage")
}

const (
	usageMessage = "" +
		`usage of 'rdb'':
	rdb [global options] <subcmd> [command specific options]

subcmd: info, get, put, delete, backup, restore, http, backup_info, purge

show subcommand help:
	rdb <subcmd> -h

global options:

`
	cmdBackup     = "backup"
	cmdBackupInfo = "backup_info"
	cmdGet        = "get"
	cmdPut        = "put"
	cmdRestore    = "restore"
	cmdPurge      = "purge"
	cmdInfo       = "info"
	cmdSet        = "set"
	cmdDelete     = "delete"
	cmdHttp       = "http"
)

func cmdUsage() {
	logger.LogI("%s", usageMessage)
	flag.PrintDefaults()
}

func (cmd *DBCmd) delete() error {
	err := cmd.initDB()
	if err != nil {
		return err
	}
	return cmd.rdb.DeleteRange(&cmd.Option.RangeOption, cmd.snk)
}

func (cmd *DBCmd) get() error {
	err := cmd.initDB()
	if err != nil {
		return err
	}
	return cmd.rdb.GetRange(&cmd.Option.RangeOption, cmd.snk)
}

func (cmd *DBCmd) parseSubCommand(args []string) error {
	var err error
	cmd.subCommand = args[0]

	switch cmd.subCommand {
	case cmdGet, cmdDelete:
		cmd.fsSubcommand.StringVar(&cmd.Option.CF, "cf", "default", "specify column family")
		cmd.fsSubcommand.StringVar(&cmd.Key, "k", "", "specify key")
		cmd.fsSubcommand.StringVar(&cmd.StartKey, "s", "", "specify start key for range operation")
		cmd.fsSubcommand.StringVar(&cmd.EndKey, "e", "", "specify end key for range operation")
		cmd.fsSubcommand.Int64Var(&cmd.StartTS, "S", 0, "specify start timestamp for range operation")
		cmd.fsSubcommand.Int64Var(&cmd.EndTS, "E", 0, "specify end timestamp for range operation")
		cmd.fsSubcommand.StringVar(&cmd.KeySeparator, "d", ",", "specify separator for key")
		cmd.fsSubcommand.IntVar(&cmd.TSFieldIndex, "i", 1, "specify the timestamp field index (start from 0) in the key")
		cmd.fsSubcommand.Int64Var(&cmd.Limit, "l", -1, "the maximum entries to return, no limitation if <= 0")
	case cmdRestore:
		cmd.fsSubcommand.StringVar(&cmd.restorePath, "t", "", "restore to directory")
		cmd.fsSubcommand.StringVar(&cmd.backupPath, "f", "", "restore from the backup directory")
	case cmdBackup:
		cmd.fsSubcommand.StringVar(&cmd.backupPath, "t", "", "backup to directory")
	case cmdHttp:
		cmd.fsSubcommand.StringVar(&cmd.addr, "l", ":9000", "http listen address")
	case cmdInfo:
	case cmdPurge:
		cmd.fsSubcommand.UintVar(&cmd.keepCnt, "k", 30, "how many old backups to keep")
	case cmdBackupInfo:
		cmd.fsSubcommand.UintVar(&cmd.keepCnt, "d", 30, "how many old backups to keep")
	default:
		break
	}

	err = cmd.fsSubcommand.Parse(args[1:])
	if err != nil {
		return fmt.Errorf("parse subcommand failed, error: %s\n", err)
	}

	if cmd.subCmdHelp {
		cmd.fsSubcommand.Usage()
		os.Exit(1)
	}

	now := time.Now().Unix()
	if cmd.StartTS < 0 {
		cmd.StartTS = now + cmd.StartTS
	}
	if cmd.EndTS < 0 {
		cmd.EndTS = now + cmd.EndTS
	}
	if cmd.TSFieldIndex < 0 {
		err = fmt.Errorf("invalid TSFieldIndex: %d", cmd.TSFieldIndex)
	}
	return err
}

func (cmd *DBCmd) backupInfo() error {
	if len(cmd.dbpath) == 0 {
		return fmt.Errorf("%s", "dbpath (-d) must be specified")
	}
	info, err := rocksdb.GetBackupInfo(cmd.dbpath)
	if err != nil {
		return err
	}
	for _, elem := range info {
		logger.Write(cmd.snk, "Backup info: %+v\n", *elem)
	}
	return nil
}

func (cmd *DBCmd) purge() error {
	return rocksdb.PurgeOldBackups(cmd.dbpath, uint32(cmd.keepCnt))
}

func (cmd *DBCmd) serveHttp() error {
	err := cmd.initDB()
	if err != nil {
		return err
	}
	return http.ListenAndServe(cmd.addr, http.HandlerFunc(cmd.rdb.HandleHttpRequest))
}

func (cmd *DBCmd) backup() error {
	err := cmd.initDB()
	if err != nil {
		return err
	}
	if len(cmd.backupPath) == 0 {
		return fmt.Errorf("-t option is required to backup")
	}
	return cmd.rdb.Backup(cmd.backupPath)
}

func (cmd *DBCmd) restore() error {
	if len(cmd.restorePath) == 0 || len(cmd.backupPath) == 0 {
		return fmt.Errorf("-f and -t option are both required to restore")
	}
	return rocksdb.Restore(cmd.backupPath, cmd.restorePath)
}

func (cmd *DBCmd) doSubCommand() error {
	var err error

	if err = helper.SetProcessUser(cmd.user); err != nil {
		return fmt.Errorf("switch to user: %s failed, %s", cmd.user, err)
	}

	if len(cmd.output) > 0 {
		if cmd.snk, err = sink.NewFile(cmd.output); err != nil {
			return err
		}
	} else {
		cmd.snk = sink.New(os.Stdout)
	}

	switch cmd.subCommand {
	case cmdInfo:
		err = cmd.dbInfo()
	case cmdGet:
		err = cmd.get()
	case cmdDelete:
		err = cmd.delete()
	case cmdRestore:
		err = cmd.restore()
	case cmdBackup:
		err = cmd.backup()
	case cmdHttp:
		err = cmd.serveHttp()
	case cmdPurge:
		err = cmd.purge()
	case cmdBackupInfo:
		err = cmd.backupInfo()
	default:
		err = fmt.Errorf("doSubCommand: %s is not implemented", cmd.subCommand)
	}
	return err
}

func (cmd *DBCmd) initDB() error {
	var err error
	if cmd.rdb, err = rocksdb.New(cmd.dbpath, cmd.readonly); err != nil {
		return err
	}
	return nil
}

func (cmd *DBCmd) dbInfo() error {
	err := cmd.initDB()
	if err != nil {
		return err
	}
	return cmd.rdb.Info(cmd.snk, cmd.verbose)
}

func main() {
	flag.Parse()
	if cmd.verbose {
		logger.SetLevel(logger.DEBUG)
	}
	if flag.NArg() == 0 {
		logger.LogW("%s", "subcommand required\n\n")
		flag.Usage()
		os.Exit(1)
	}

	var err error
	if err = cmd.parseSubCommand(flag.Args()); err != nil {
		goto out
	}
	err = cmd.doSubCommand()
out:
	if err != nil {
		if cmd.snk == nil {
			cmd.snk = sink.New(os.Stderr)
		}
		io.WriteString(cmd.snk, fmt.Sprintf(`{"error":"%s"}`, err))
		io.WriteString(cmd.snk, "\n")
	}
}
