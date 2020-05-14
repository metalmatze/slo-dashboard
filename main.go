package main

import (
	"context"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/prometheus/client_golang/api"
	prometheusv1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/urfave/cli"
)

type configuration struct {
	PrometheusURL string        `json:"prometheus_url"`
	Applications  []Application `json:"applications"`
	//Projects      []struct {
	//	Name          string `json:"name"`
	//	Days          uint   `json:"days"`
	//	PrometheusURL string `json:"prometheus_url"`
	//	Objective     struct {
	//		Target float64 `json:"target"`
	//		Query  string  `json:"query"`
	//	} `json:"objective"`
	//	Data []struct {
	//		Title string `json:"title"`
	//		SLO   struct {
	//			Value      float64 `json:"value"`
	//			Comparison string  `json:"comparison"`
	//		} `json:"slo"`
	//		Format string `json:"format"`
	//		Query  string `json:"query"`
	//	} `json:"data"`
	//} `json:"projects"`
}

type Application struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Components  []struct {
		Name   string `json:"name"`
		Status struct {
			Degraded struct {
				Query string `json:"query"`
			} `json:"degraded"`
			Outage struct {
				Query string `json:"query"`
			} `json:"outage"`
		} `json:"status"`
	} `json:"components"`
}

type Project struct {
	Name          string
	PrometheusURL string
	Objective     Objective
	Dates         []time.Time
	Columns       []Column
}

type Objective struct {
	Value  float64
	Target float64
	Days   uint
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

	_ = prometheusv1.NewAPI(client)

	//for _, p := range config.Projects {
	//	for _, d := range p.Data {
	//		value, _, err := promAPI.Query(context.TODO(), d.Query, time.Now())
	//		if err != nil {
	//			return fmt.Errorf("failed to run query: %w", err)
	//		}
	//
	//		vec := value.(model.Vector)
	//
	//		for _, v := range vec {
	//			fmt.Printf("%.2f\n", v.Value)
	//		}
	//	}
	//}

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

	_, err = query(config)
	if err != nil {
		return fmt.Errorf("failed querying: %w", err)
	}

	return nil

	//return serve(projects)
}

func query(config configuration) ([]Application, error) {
	var applications []Application

	for _, application := range config.Applications {
		client, err := api.NewClient(api.Config{Address: config.PrometheusURL})
		if err != nil {
			return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
		}
		promAPI := prometheusv1.NewAPI(client)

		for _, component := range application.Components {
			value, _, err := promAPI.Query(context.TODO(), component.Status.Degraded.Query, time.Now())
			if err != nil {
				return nil, nil
			}

			var result float64
			vec := value.(model.Vector)
			for _, v := range vec {
				result = float64(v.Value)
			}

			fmt.Printf("%s - %s - degraded: %v\n", application.Name, component.Name, result == 1.0)

			value, _, err = promAPI.Query(context.TODO(), component.Status.Outage.Query, time.Now())
			if err != nil {
				return nil, fmt.Errorf("failed to query Prometheus: %v", err)
			}

			vec = value.(model.Vector)
			for _, v := range vec {
				result = float64(v.Value)
			}

			fmt.Printf("%s - %s - outage: %v\n", application.Name, component.Name, result == 1.0)
		}

	}

	return applications, nil

	//for _, cp := range config.Projects {
	//	var prometheusURL string
	//	if cp.PrometheusURL != "" {
	//		prometheusURL = cp.PrometheusURL // use project's Prometheus URL
	//	} else if config.PrometheusURL != "" {
	//		prometheusURL = config.PrometheusURL // use global Prometheus URL
	//	} else {
	//		return nil, fmt.Errorf("no Prometheus URL found")
	//	}
	//
	//	client, err := api.NewClient(api.Config{Address: prometheusURL})
	//	if err != nil {
	//		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	//	}
	//	promAPI := prometheusv1.NewAPI(client)
	//
	//	project := Project{Name: cp.Name}
	//
	//	objective, err := queryObjective(promAPI, cp.Objective.Query, cp.Days)
	//	if err != nil {
	//		return nil, err
	//	}
	//
	//	project.Objective = Objective{
	//		Value:  objective,
	//		Target: cp.Objective.Target,
	//		Days:   cp.Days,
	//	}
	//
	//	fmt.Println("budget", time.Duration((1-project.Objective.Target)*3600*24*float64(project.Objective.Days))*time.Second)
	//	fmt.Println("budget left", time.Duration((project.Objective.Value-project.Objective.Target)*3600*24*float64(project.Objective.Days))*time.Second)
	//
	//	projects = append(projects, project)
	//
	//	continue
	//
	//	// This is basically midnight just one second less to still have the date of the previous day
	//	year, month, day := time.Now().Date()
	//	midnight := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Add(-1 * time.Second)
	//
	//	for i := 0; i < int(cp.Days); i++ {
	//		project.Dates = append(project.Dates, midnight.AddDate(0, 0, -i))
	//	}
	//
	//	for _, d := range cp.Data {
	//		c := Column{Title: d.Title, SLO: SLO{
	//			Value:      d.SLO.Value,
	//			Comparison: d.SLO.Comparison,
	//		}}
	//
	//		fmt.Println(c.Title)
	//
	//		switch d.Format {
	//		case string(seconds):
	//			c.Format = seconds
	//		case string(percentage):
	//			c.Format = percentage
	//		case string(percentage100):
	//			c.Format = percentage100
	//		}
	//
	//		for i := 0; i < int(cp.Days); i++ {
	//			day := midnight.AddDate(0, 0, -i)
	//			fmt.Printf("\t%s\n", day.Format("2006-01-02"))
	//
	//			query := strings.Replace(d.Query, "$interval", "24h", -1)
	//
	//			value, _, err := promAPI.Query(context.TODO(), query, day)
	//			if err != nil {
	//				return nil, fmt.Errorf("querying failed: %w", err)
	//			}
	//
	//			var result float64
	//			vec := value.(model.Vector)
	//			for _, v := range vec {
	//				result = float64(v.Value)
	//			}
	//			c.Data = append(c.Data, result)
	//		}
	//		project.Columns = append(project.Columns, c)
	//	}
	//	projects = append(projects, project)
	//}
	//return , nil
}

func queryObjective(api prometheusv1.API, query string, days uint) (float64, error) {
	q := prepareQuery(query, fmt.Sprintf("%dd", days))

	value, _, err := api.Query(context.TODO(), q, time.Now())
	if err != nil {
		return 0, err
	}

	var result float64
	vec := value.(model.Vector)
	for _, v := range vec {
		result = float64(v.Value)
	}
	return result, nil
}

func prepareQuery(q string, interval string) string {
	return strings.Replace(q, "$interval", interval, -1)
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
