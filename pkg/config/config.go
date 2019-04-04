package config

import (
	"fmt"
	"os"
	"path"
	"time"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const currentApiVersion = "descheduler.lentil1016.cn/v1alpha1"

type config struct {
	apiVersion string     `yaml: apiVersion`
	spec       ConfigSpec `yaml: spec`
}

type ConfigSpec struct {
	KubeConfigFile string         `yaml: kubeconfig`
	DryRun         bool           `yaml: dryRun`
	Triggers       ConfigTriggers `yaml: triggers`
	Rules          ConfigRules    `yaml: rules`
}

type ConfigTriggers struct {
	AllReplicasOnOneNode bool                     `yaml: allReplicasOnOneNode` // If all pods(more than one) of a replicaSet run on one single node, evict one of them.
	MinSparedPercentage  ConfigResourcePercentage `yaml: minSparedPercentage`
	MaxSparedPercentage  ConfigResourcePercentage `yaml: maxSparedPercentage`
	Mode                 string                   `yaml: mode`
	Time                 ConfigTime               `yaml: time`
}

type ConfigResourcePercentage struct {
	CPU    float64 `yaml: cpu`
	Memory float64 `yaml: memory`
	Pod    float64 `yaml: pod`
}

type ConfigTime struct {
	From time.Time `yaml: from`
	For  string    `yaml: for`
}

type ConfigRules struct {
	HardEviction     bool     `yaml: hardEviction`     // Evicting a pod when it's the only replica of the replicaSet it belongs.
	AffectNamespaces []string `yaml: affectNamespaces` // Namespaces that descheduler will affect to, an empty slice indicates all namespaces
	NodeSelector     string   `yaml: nodeSelector`     // Selectors of the nodes that descheduler will affect to, nil indicates all nodes.
	MaxEvictSize     int      `yaml: maxEvictSize`     // Number of the Pod in one deschedule term will be evicted at most.
}

func setDefaults() {
	var defaultFromTime = time.Now()

	var defaultConf = ConfigSpec{
		DryRun: false,
		Triggers: ConfigTriggers{
			AllReplicasOnOneNode: true,
			MinSparedPercentage: ConfigResourcePercentage{
				CPU:    30,
				Memory: 30,
				Pod:    30,
			},
			MaxSparedPercentage: ConfigResourcePercentage{
				CPU:    70,
				Memory: 70,
				Pod:    70,
			},
			Mode: "event",
			Time: ConfigTime{
				From: defaultFromTime,
				For:  "1h",
			},
		},
		Rules: ConfigRules{
			HardEviction:     false,
			AffectNamespaces: []string{},
			NodeSelector:     "",
			MaxEvictSize:     3,
		},
	}

	viper.SetDefault("spec.dryRun", defaultConf.DryRun)
	viper.SetDefault("spec.triggers.preventAllReplicasOnOneNode", defaultConf.Triggers.AllReplicasOnOneNode)
	viper.SetDefault("spec.triggers.minSparedPercentage.cpu", defaultConf.Triggers.MinSparedPercentage.CPU)
	viper.SetDefault("spec.triggers.minSparedPercentage.memory", defaultConf.Triggers.MinSparedPercentage.Memory)
	viper.SetDefault("spec.triggers.minSparedPercentage.pod", defaultConf.Triggers.MinSparedPercentage.Pod)
	viper.SetDefault("spec.triggers.maxSparedPercentage.cpu", defaultConf.Triggers.MaxSparedPercentage.CPU)
	viper.SetDefault("spec.triggers.maxSparedPercentage.memory", defaultConf.Triggers.MaxSparedPercentage.Memory)
	viper.SetDefault("spec.triggers.maxSparedPercentage.pod", defaultConf.Triggers.MaxSparedPercentage.Pod)
	viper.SetDefault("spec.triggers.mode", defaultConf.Triggers.Mode)
	viper.SetDefault("spec.triggers.time.from", defaultConf.Triggers.Time.From)
	viper.SetDefault("spec.triggers.time.for", defaultConf.Triggers.Time.For)
	viper.SetDefault("spec.rules.hardEviction", defaultConf.Rules.HardEviction)
	viper.SetDefault("spec.rules.affectNamespaces", defaultConf.Rules.AffectNamespaces)
	viper.SetDefault("spec.rules.nodeSelector", defaultConf.Rules.NodeSelector)
	viper.SetDefault("spec.rules.maxEvictSize", defaultConf.Rules.MaxEvictSize)
}

func InitConfig(configFile string, kubeConfigFile string, dryRun bool) {
	// Find home directory.
	home, err := homedir.Dir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// set config file path
	if configFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(configFile)
	} else {
		// Search config in home directory with name ".descheduler" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".descheduler")
	}
	viper.SetDefault("spec.kubeconfig", path.Join(home, ".kube/config"))
	setDefaults()
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
	// If config file have a wrong apiVersion, reset config to default
	if viper.GetString("apiVersion") != currentApiVersion {
		fmt.Printf("Error apiVersion %v in config file %v, expecting for %v. will use the default config\n", viper.GetString("apiVersion"), viper.ConfigFileUsed(), currentApiVersion)
		viper.Reset()
	}

	// Cmdline param overrides config file contents
	if kubeConfigFile != "" {
		viper.Set("spec.kubeconfig", kubeConfigFile)
	}
	if dryRun == true {
		viper.Set("spec.dryRun", dryRun)
	}
}

func GetConfig() ConfigSpec {
	// Create config with values in viper
	return ConfigSpec{
		KubeConfigFile: viper.GetString("spec.kubeconfig"),
		DryRun:         viper.GetBool("spec.dryRun"),
		Triggers: ConfigTriggers{
			AllReplicasOnOneNode: viper.GetBool("spec.triggers.allReplicasOnOneNode"),
			MinSparedPercentage: ConfigResourcePercentage{
				CPU:    viper.GetFloat64("spec.triggers.minSparedPercentage.cpu"),
				Memory: viper.GetFloat64("spec.triggers.minSparedPercentage.memory"),
				Pod:    viper.GetFloat64("spec.triggers.minSparedPercentage.pod"),
			},
			MaxSparedPercentage: ConfigResourcePercentage{
				CPU:    viper.GetFloat64("spec.triggers.maxSparedPercentage.cpu"),
				Memory: viper.GetFloat64("spec.triggers.maxSparedPercentage.memory"),
				Pod:    viper.GetFloat64("spec.triggers.maxSparedPercentage.pod"),
			},
			Mode: viper.GetString("spec.triggers.mode"),
			Time: ConfigTime{
				From: viper.GetTime("spec.triggers.time.from"),
				For:  viper.GetString("spec.triggers.time.for"),
			},
		},
		Rules: ConfigRules{
			HardEviction:     viper.GetBool("spec.rules.hardEviction"),
			AffectNamespaces: viper.GetStringSlice("spec.rules.affectNamespaces"),
			NodeSelector:     viper.GetString("spec.rules.nodeSelector"),
			MaxEvictSize:     viper.GetInt("spec.rules.maxEvictSize"),
		},
	}
}
