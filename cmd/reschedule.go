package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lentil1016/descheduler/pkg/descheduler"
	"github.com/spf13/cobra"
)

// doDescheduleCmd do the calculate and then deschedule
func doDescheduleCmd(cmd *cobra.Command, args []string) {
	d, err := descheduler.CreateDescheduler()
	if err != nil {
		fmt.Println(err)
	}
	stopCh := make(chan struct{})
	defer close(stopCh)
	go d.Run(stopCh)

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	signal.Notify(sigterm, syscall.SIGINT)
	<-sigterm
}
