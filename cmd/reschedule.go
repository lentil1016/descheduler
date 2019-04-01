package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/lentil1016/descheduler/pkg/client"
	"github.com/lentil1016/descheduler/pkg/descheduler"
	"github.com/spf13/cobra"
)

// doDescheduleCmd do the calculate and then deschedule
func doDescheduleCmd(cmd *cobra.Command, args []string) {
	client, err := client.CreateClient(kubeConfigFile)
	if client == nil {
		fmt.Println(err)
		return
	}

	d, err := descheduler.CreateDescheduler(client)
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
