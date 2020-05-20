package plugin

import (
	"context"
	"errors"
	"time"

	"github.com/hashicorp/go-plugin"
)

type ReattachConfig struct {
	Protocol string
	Pid      int
	Test     bool
	Addr     ReattachConfigAddr
}

type ReattachConfigAddr struct {
	Network string
	String  string
}

func DebugServe(ctx context.Context, opts *ServeOpts) (ReattachConfig, <-chan struct{}, error) {
	reattachCh := make(chan *plugin.ReattachConfig)
	closeCh := make(chan struct{})

	opts.TestConfig = &plugin.ServeTestConfig{
		Context:          ctx,
		ReattachConfigCh: reattachCh,
		CloseCh:          closeCh,
	}

	go Serve(opts)

	var config *plugin.ReattachConfig
	select {
	case config = <-reattachCh:
	case <-time.After(2 * time.Second):
		return ReattachConfig{}, closeCh, errors.New("timeout waiting on reattach config")
	}

	if config == nil {
		return ReattachConfig{}, closeCh, errors.New("nil reattach config received")
	}

	return ReattachConfig{
		Protocol: string(config.Protocol),
		Pid:      config.Pid,
		Test:     config.Test,
		Addr: ReattachConfigAddr{
			Network: config.Addr.Network(),
			String:  config.Addr.String(),
		},
	}, closeCh, nil
}
