package steward

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml"
)

type Configuration struct {
	// The configuration folder on disk
	ConfigFolder string
	// some unique string to identify this Edge unit
	NodeName string
	// the address of the message broker
	BrokerAddress string
	// The number of the profiling port
	ProfilingPort string
	// host and port for prometheus listener, e.g. localhost:2112
	PromHostAndPort string
	// set to true if this is the node that should receive the error log's from other nodes
	CentralErrorLogger bool
	// default message timeout in seconds. This can be overridden on the message level")
	DefaultMessageTimeout int
	// default amount of retries that will be done before a message is thrown away, and out of the system
	DefaultMessageRetries int
	// Make the current node send hello messages to central at given interval in seconds
	PublisherServiceSayhello int
	// Publisher data folder
	SubscribersDataFolder string
}

func NewConfiguration() *Configuration {
	c := Configuration{}
	return &c
}

func newConfigurationDefaults() Configuration {
	c := Configuration{
		ConfigFolder:             "./etc",
		BrokerAddress:            "127.0.0.1:4222",
		ProfilingPort:            "",
		PromHostAndPort:          "",
		CentralErrorLogger:       false,
		DefaultMessageTimeout:    10,
		DefaultMessageRetries:    1,
		PublisherServiceSayhello: 30,
		SubscribersDataFolder:    "./data",
	}
	return c
}

func (c *Configuration) CheckFlags() {
	// TODO: Look into how to be able to specify the location via flag,
	// now it is only using the default location which is /etc.
	// The problem is that the flags are parsed after the default config
	// is set, so we need a way to get that location out of the flag when
	// creating the config file the first time.

	// Create an empty default config
	var fc Configuration

	// Read file config. Set system default if it can't find config file.
	fc, err := c.ReadConfigFile()
	if err != nil {
		log.Printf("%v\n", err)
		fc = newConfigurationDefaults()
	}

	flag.StringVar(&c.ConfigFolder, "configFolder", fc.ConfigFolder, "folder who contains the config file")
	flag.StringVar(&c.NodeName, "node", fc.NodeName, "some unique string to identify this Edge unit")
	flag.StringVar(&c.BrokerAddress, "brokerAddress", fc.BrokerAddress, "the address of the message broker")
	flag.StringVar(&c.ProfilingPort, "profilingPort", fc.ProfilingPort, "The number of the profiling port")
	flag.StringVar(&c.PromHostAndPort, "promHostAndPort", fc.PromHostAndPort, "host and port for prometheus listener, e.g. localhost:2112")
	flag.BoolVar(&c.CentralErrorLogger, "centralErrorLogger", fc.CentralErrorLogger, "set to true if this is the node that should receive the error log's from other nodes")
	flag.IntVar(&c.DefaultMessageTimeout, "defaultMessageTimeout", fc.DefaultMessageTimeout, "default message timeout in seconds. This can be overridden on the message level")
	flag.IntVar(&c.DefaultMessageRetries, "defaultMessageRetries", fc.DefaultMessageRetries, "default amount of retries that will be done before a message is thrown away, and out of the system")
	flag.IntVar(&c.PublisherServiceSayhello, "publisherServiceSayhello", fc.PublisherServiceSayhello, "Make the current node send hello messages to central at given interval in seconds")
	flag.StringVar(&c.SubscribersDataFolder, "subscribersDataFolder", fc.SubscribersDataFolder, "The data folder where subscribers are allowed to write their data if needed")

	flag.Parse()

	if err := c.WriteConfigFile(); err != nil {
		log.Printf("error: checkFlags: failed writing config file: %v\n", err)
		os.Exit(1)
	}
}

func (c *Configuration) ReadConfigFile() (Configuration, error) {
	fp := filepath.Join("./etc/", "config.toml")

	if _, err := os.Stat(fp); os.IsNotExist(err) {
		return Configuration{}, fmt.Errorf("error: no config file found %v: %v", fp, err)
	}

	f, err := os.OpenFile(fp, os.O_RDONLY, 0600)
	if err != nil {
		return Configuration{}, fmt.Errorf("error: ReadConfigFile: failed to open file: %v", err)
	}
	defer f.Close()

	var conf Configuration
	dec := toml.NewDecoder(f)
	err = dec.Decode(&conf)
	if err != nil {
		return Configuration{}, fmt.Errorf("error: decode toml file %v: %v", fp, err)
	}

	fmt.Printf("%+v\n", c)
	return conf, nil
}

// WriteConfigFile will write the current config to file. If the file or the
// directory for the config file does not exist it will be created.
func (c *Configuration) WriteConfigFile() error {
	if _, err := os.Stat(c.ConfigFolder); os.IsNotExist(err) {
		err := os.Mkdir(c.ConfigFolder, 0700)
		if err != nil {
			return fmt.Errorf("error: failed to create directory %v: %v", c.ConfigFolder, err)
		}
	}

	fp := filepath.Join(c.ConfigFolder, "config.toml")

	f, err := os.OpenFile(fp, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("error: WriteConfigFile: failed to open file: %v", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	enc.Encode(c)

	return nil
}