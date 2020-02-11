package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/urfave/cli"
)

func main() {
	app := cli.NewApp()
	app.Name = "slo-dashboard"

	app.Action = action
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "file,f",
			Usage: "Path to the configuration file",
			Value: "projects.yaml",
		},
		cli.StringFlag{
			Name:  "prometheus.url",
			Usage: "The full URL to connect to Prometheus with",
			Value: "http://localhost:9090/api/v1",
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Println(err)
	}
}

func action(c *cli.Context) error {
	config := struct {
		Projects []struct {
			Name string `json:"name"`
			Data []struct {
				Name  string `json:"name"`
				Query string `json:"query"`
			} `json:"data"`
		} `json:"projects"`
	}{}

	bytes, err := ioutil.ReadFile(c.String("file"))
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	client, err := api.NewClient(api.Config{
		Address: c.String("prometheus.url"),
	})
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	promAPI := prometheusv1.NewAPI(client)

	for _, p := range config.Projects {
		for _, d := range p.Data {
			value, _, err := promAPI.Query(context.TODO(), d.Query, time.Now())
			if err != nil {
				return fmt.Errorf("failed to run query: %w", err)
			}

			vec := value.(model.Vector)

			for _, v := range vec {
				fmt.Printf("%.2f\n", v.Value)
			}
		}
	}

	return nil
}
