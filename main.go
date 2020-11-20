package main

import (
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gopkg.in/yaml.v2"
)

var (
	a   *oracle
	cfg *config
)

func init() {
	confPath := os.Getenv("ORACLE_CONFPATH")

	if confPath == "" {
		confPath = "./etc/config.yml"
	}

	yamlFile, err := ioutil.ReadFile(confPath)

	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(yamlFile, &cfg)

	if err != nil {
		log.Fatal(err)
	}

	err = initOracle()

	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	err := a.Run()

	if err != nil {
		log.Fatal(err)
	}

	log.Println("Success start")

	listenSignals()
}

func listenSignals() {
	signals := make(chan os.Signal, 1)

	signal.Notify(signals,
		syscall.SIGINT,
		syscall.SIGTERM,
	)

	for sig := range signals {
		log.Println("Got signal: " + sig.String())

		_ = destroyOracle()

		return
	}
}

func initOracle() error {
	if a != nil {
		return nil
	}

	var err error
	a, err = newOracle()

	if err != nil {
		return err
	}

	return nil
}

func destroyOracle() error {
	if a != nil {
		a.Stop()
		a = nil
	}

	return nil
}
