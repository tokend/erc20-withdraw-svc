package cli

import (
	"context"
	"fmt"
	"gitlab.com/distributed_lab/kit/kv"
	"gitlab.com/distributed_lab/logan/v3"
	"github.com/tokend/erc20-withdraw-svc/internal/config"
	"github.com/tokend/erc20-withdraw-svc/internal/services/withdrawer"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

func Run(args []string) bool {
	log := logan.New()

	defer func() {
		if rvr := recover(); rvr != nil {
			log.WithRecover(rvr).Error("app panicked")
		}
	}()

	app := kingpin.New("erc20-withdraw-svc", "")
	runCmd := app.Command("run", "run command")
	withdraw := runCmd.Command("withdraw", "run withdraw service")
	versionCmd := app.Command("version", "service revision")

	cfg := config.NewConfig(kv.MustFromEnv())
	log = cfg.Log()

	cmd, err := app.Parse(args[1:])
	if err != nil {
		log.WithError(err).Error("failed to parse arguments")
	}

	switch cmd {
	case withdraw.FullCommand():
		svc := withdrawer.New(cfg)
		svc.Run(context.Background())
	case versionCmd.FullCommand():
		fmt.Println(config.ERC20WithdrawVersion)
	default:
		log.Errorf("unknown command %s", cmd)
		return false
	}

	return true
}
