package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/urfave/cli"
)

type configuration struct {
	PrometheusURL string `json:"prometheus_url"`
	Projects      []struct {
		Name          string `json:"name"`
		Days          uint   `json:"days"`
		PrometheusURL string `json:"prometheus_url"`
		Data          []struct {
			Title string `json:"title"`
			SLO   struct {
				Value      float64 `json:"value"`
				Comparison string  `json:"comparison"`
			} `json:"slo"`
			Format string `json:"format"`
			Query  string `json:"query"`
		} `json:"data"`
	} `json:"projects"`
}

type Project struct {
	Name          string
	PrometheusURL string
	Dates         []time.Time
	Columns       []Column
}

type Column struct {
	Title  string
	SLO    SLO
	Format format
	Data   []float64
}
type SLO struct {
	Value      float64
	Comparison string
}

type format string

const (
	seconds       format = "seconds"
	percentage    format = "percentage"
	percentage100 format = "percentage100"
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
	}

	app.Commands = []cli.Command{
		{
			Name:   "volatile",
			Usage:  "Volatile means that all queries are run at start and only kept in-memory until shutdown",
			Action: volatile,
			Flags: []cli.Flag{
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
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Println(err)
	}
}

func action(c *cli.Context) error {
	bytes, err := ioutil.ReadFile(c.String("file"))
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config configuration
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

func volatile(c *cli.Context) error {
	bytes, err := ioutil.ReadFile(c.String("file"))
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	var config configuration
	if err := yaml.Unmarshal(bytes, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	projects, err := query(config)
	if err != nil {
		return fmt.Errorf("failed querying: %w", err)
	}

	return serve(projects)
}

func query(config configuration) ([]Project, error) {
	var projects []Project

	for _, cp := range config.Projects {
		var prometheusURL string
		if cp.PrometheusURL != "" {
			prometheusURL = cp.PrometheusURL // use project's Prometheus URL
		} else if config.PrometheusURL != "" {
			prometheusURL = config.PrometheusURL // use global Prometheus URL
		} else {
			return nil, fmt.Errorf("no Prometheus URL found")
		}

		client, err := api.NewClient(api.Config{Address: prometheusURL})
		if err != nil {
			return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
		}
		promAPI := prometheusv1.NewAPI(client)

		project := Project{Name: cp.Name}

		// This is basically midnight just one second less to still have the date of the previous day
		year, month, day := time.Now().Date()
		midnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Add(-1 * time.Second)

		for i := 0; i < int(cp.Days); i++ {
			project.Dates = append(project.Dates, midnight.AddDate(0, 0, -i))
		}

		for _, d := range cp.Data {
			c := Column{Title: d.Title, SLO: SLO{
				Value:      d.SLO.Value,
				Comparison: d.SLO.Comparison,
			}}

			fmt.Println(c.Title)

			switch d.Format {
			case string(seconds):
				c.Format = seconds
			case string(percentage):
				c.Format = percentage
			case string(percentage100):
				c.Format = percentage100
			}

			for i := 0; i < int(cp.Days); i++ {
				day := midnight.AddDate(0, 0, -i)
				fmt.Printf("\t%s\n", day.Format("2006-01-02"))

				value, _, err := promAPI.Query(context.TODO(), d.Query, day)
				if err != nil {
					return nil, fmt.Errorf("querying failed: %w", err)
				}

				var result float64
				vec := value.(model.Vector)
				for _, v := range vec {
					result = float64(v.Value)
				}
				c.Data = append(c.Data, result)
			}
			project.Columns = append(project.Columns, c)
		}
		projects = append(projects, project)
	}
	return projects, nil
}

func serve(projects []Project) error {
	funcs := map[string]interface{}{
		"datefmt": func(t time.Time) string {
			return t.Format("2006-01-02")
		},
		"format": func(format format, v float64) string {
			switch format {
			case seconds:
				d := time.Duration(v * float64(time.Second))

				if d.Hours() >= 1 {
					return fmt.Sprintf("%.fh", d.Hours())
				}
				if d.Minutes() >= 1 {
					return fmt.Sprintf("%.fm", d.Minutes())
				}
				if d.Seconds() >= 1 {
					return fmt.Sprintf("%.fs", d.Seconds())
				}
				if d.Milliseconds() >= 1 {
					return fmt.Sprintf("%dms", d.Milliseconds())
				}
				if d.Microseconds() >= 1 {
					return fmt.Sprintf("%dµs", d.Microseconds())
				}
				if d.Nanoseconds() >= 1 {
					return fmt.Sprintf("%dns", d.Nanoseconds())
				}
				return "0s"
			case percentage:
				return fmt.Sprintf("%.2f%%", v*100)
			}

			return fmt.Sprintf("%.f", v)
		},
		"breach": func(slo SLO, value float64) bool {
			if slo.Value == 0 { // No SLO
				return false
			}

			// We negate as we look for breaches of SLO here
			// This could be simplified but left like this for readability
			switch slo.Comparison {
			case "higher":
				return !(value > slo.Value)
			case "lower":
				return !(value < slo.Value)
			}

			return false
		},
	}

	tmpl, err := template.New("index.html").Funcs(funcs).ParseFiles("./html/index.html")
	if err != nil {
		return err
	}

	data := struct {
		Projects []Project
	}{Projects: projects}

	m := &http.ServeMux{}
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})

	s := http.Server{Addr: ":8080", Handler: m}

	fmt.Println("Running HTTP server on address", s.Addr)
	return s.ListenAndServe()
}
