package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

const currentApiVersion = "descheduler.lentil1016.cn/v1alpha1"

type config struct {
	apiVersion string     `yaml: apiVersion`
	spec       ConfigSpec `yaml: spec`
}

type ConfigSpec struct {
	Triggers ConfigTriggers `yaml: tiggers`
	Rules    ConfigRules    `yaml: rules`
}

type ConfigTriggers struct {
	AllReplicasOnOneNode bool                     `yaml: allReplicasOnOneNode` // If all pods(more than one) of a replicaSet run on one single node, evict one of them.
	MaxGiniPercentage    ConfigResourcePercentage `yaml: maxGiniPercentage`
	MaxSparedPercentage  ConfigResourcePercentage `yaml: maxSparedPercentage`
	On                   string                   `yaml: on`
	Time                 ConfigTime               `yaml: time`
}

type ConfigResourcePercentage struct {
	CPU    int `yaml: cpu`
	Memory int `yaml: memory`
}

type ConfigTime struct {
	From time.Time `yaml: from`
	For  string    `yaml: for`
}

type ConfigRules struct {
	HardEviction      bool     `yaml: hardEviction`      // Evicting a pod when it's the only replica of the replicaSet it belongs.
	WorkingNamespaces []string `yaml: workingNamespaces` // Namespaces that descheduler will affect to, an empty slice indicates all namespaces
	NodeSelector      string   `yaml: nodeSelector`      // Selectors of the nodes that descheduler will affect to, nil indicates all nodes.
}

var defaultFromTime, _ = time.Parse("11:00PM", "11:00PM")

var defaultConf = ConfigSpec{
	Triggers: ConfigTriggers{
		AllReplicasOnOneNode: true,
		MaxGiniPercentage: ConfigResourcePercentage{
			CPU:    50,
			Memory: 50,
		},
		MaxSparedPercentage: ConfigResourcePercentage{
			CPU:    80,
			Memory: 80,
		},
		On: "event",
		Time: ConfigTime{
			From: defaultFromTime,
			For:  "1h",
		},
	},
	Rules: ConfigRules{
		HardEviction:      false,
		WorkingNamespaces: []string{},
		NodeSelector:      "",
	},
}

func setDefaults() {
	viper.SetDefault("spec.triggers.preventAllReplicasOnOneNode", defaultConf.Triggers.AllReplicasOnOneNode)
	viper.SetDefault("spec.triggers.maxGiniPercentage.cpu", defaultConf.Triggers.MaxGiniPercentage.CPU)
	viper.SetDefault("spec.triggers.maxGiniPercentage.memory", defaultConf.Triggers.MaxGiniPercentage.Memory)
	viper.SetDefault("spec.triggers.maxSparedPercentage.cpu", defaultConf.Triggers.MaxSparedPercentage.CPU)
	viper.SetDefault("spec.triggers.maxSparedPercentage.memory", defaultConf.Triggers.MaxSparedPercentage.Memory)
	viper.SetDefault("spec.triggers.on", defaultConf.Triggers.On)
	viper.SetDefault("spec.triggers.time.from", defaultConf.Triggers.Time.From)
	viper.SetDefault("spec.triggers.time.for", defaultConf.Triggers.Time.For)
	viper.SetDefault("spec.rules.hardEviction", defaultConf.Rules.HardEviction)
	viper.SetDefault("spec.rules.affectNamespaces", defaultConf.Rules.WorkingNamespaces)
	viper.SetDefault("spec.rules.nodeSelector", defaultConf.Rules.NodeSelector)
}

func InitConfig() {
	setDefaults()
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	if viper.GetString("apiVersion") != currentApiVersion {
		fmt.Printf("Error apiVersion %v in config file %v, expecting for %v\n", viper.GetString("apiVersion"), viper.ConfigFileUsed(), currentApiVersion)
		viper.Reset()
	}
	fmt.Println(viper.GetTime("spec.tiggers.time.from"))
	fmt.Println(time.ParseDuration(viper.GetString("spec.tiggers.time.for")))
}

func GetConfig() ConfigSpec {
	return ConfigSpec{
		Triggers: ConfigTriggers{
			AllReplicasOnOneNode: viper.GetBool("spec.triggers.allReplicasOnOneNode"),
			MaxGiniPercentage: ConfigResourcePercentage{
				CPU:    viper.GetInt("spec.triggers.maxGiniPercentage.cpu"),
				Memory: viper.GetInt("spec.triggers.maxGiniPercentage.memory"),
			},
			MaxSparedPercentage: ConfigResourcePercentage{
				CPU:    viper.GetInt("spec.triggers.maxSparedPercentage.cpu"),
				Memory: viper.GetInt("spec.triggers.maxSparedPercentage.memory"),
			},
			On: viper.GetString("spec.triggers.on"),
			Time: ConfigTime{
				From: viper.GetTime("spec.triggers.time.from"),
				For:  viper.GetString("spec.triggers.time.for"),
			},
		},
		Rules: ConfigRules{
			HardEviction:      viper.GetBool("spec.rules.hardEviction"),
			WorkingNamespaces: viper.GetStringSlice("spec.rules.workingNamespaces"),
			NodeSelector:      viper.GetString("spec.rules.nodeSelector"),
		},
	}
}
