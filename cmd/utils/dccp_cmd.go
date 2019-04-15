// Copyright 2018 The xvalue-dccp


package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xvalue/go-xvalue/internal/debug"
	"github.com/xvalue/go-xvalue/log"
	"github.com/xvalue/go-xvalue/node"
)

func StartDccpNode(stack *node.Node) {
	if err := stack.DccpStart(); err != nil {
		Fatalf("Error starting protocol stack: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go stack.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
			}
		}
		debug.Exit() // ensure trace and CPU profile data is flushed.
		debug.LoudPanic("boom")
	}()
}
