package main

import (
	"context"
	"flag"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/MixinNetwork/mixin/logger"
	"github.com/MixinNetwork/nfo/mtg"
	"github.com/MixinNetwork/tip/messenger"
	"github.com/MixinNetwork/trusted-group/mvm/config"
	"github.com/MixinNetwork/trusted-group/mvm/machine"
	"github.com/MixinNetwork/trusted-group/mvm/quorum"
	"github.com/MixinNetwork/trusted-group/mvm/store"
)

func main() {
	logger.SetLevel(logger.VERBOSE)
	ctx := context.Background()

	bp := flag.String("d", "~/.mixin/mvm/data", "database directory path")
	cp := flag.String("c", "~/.mixin/mvm/config.toml", "configuration file path")
	flag.Parse()

	if strings.HasPrefix(*cp, "~/") {
		usr, _ := user.Current()
		*cp = filepath.Join(usr.HomeDir, (*cp)[2:])
	}
	conf, err := config.ReadConfiguration(*cp)
	if err != nil {
		panic(err)
	}

	if strings.HasPrefix(*bp, "~/") {
		usr, _ := user.Current()
		*bp = filepath.Join(usr.HomeDir, (*bp)[2:])
	}
	db, err := store.OpenBadger(ctx, *bp)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	group, err := mtg.BuildGroup(ctx, db, conf.MTG)
	if err != nil {
		panic(err)
	}

	messenger, err := messenger.NewMixinMessenger(ctx, conf.Messenger)
	if err != nil {
		panic(err)
	}
	en, err := quorum.Boot()
	if err != nil {
		panic(err)
	}
	im, err := machine.Boot(conf.Machine, group, db, messenger)
	if err != nil {
		panic(err)
	}
	im.AddEngine(machine.ProcessPlatformQuorum, en)
	go im.Loop(ctx)

	group.AddWorker(im)
	group.Run(ctx)
}
