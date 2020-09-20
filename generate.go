/*

Copyright 2020 Luigi Zuccarelli

This program is free software; you can redistribute it and/or
modify it under the terms of the GNU General Public License
version 2 as published by the Free Software Foundation.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU General Public License for more details.

*/

package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/microlib/simple"
)

var (
	config      string
	stepsConfig string
)

type GenerateSchema struct {
	Organization  string
	Project       string
	Items         []AppDetails
	Time          string
	Repos         RepoDetails
	Slack         SlackDetails
	Smtp          SmtpDetails
	RegistryUrl   string
	SonarqubeUrl  string
	ProductionUrl string
	Path          string
	ReadPath      string
	Env           string
	Logger        *simple.Logger
}

type AppDetails struct {
	Project     string
	Application string
	Repo        string
	Env         string
}

type RepoDetails struct {
	Project string
	Cicd    string
	GitBase string
	User    string
	Pwd     string
}

type SlackDetails struct {
	Url     string
	Channel string
}

type SmtpDetails struct {
	Sender    string
	Recipient string
	Url       string
	Port      string
	User      string
	Password  string
	Tls       string
}

type Secrets struct {
	ImagePull string
}

type Steps struct {
	Project string
	Items   []StepItems
}

type StepItems struct {
	Name string
}

func makeDirs(schema *GenerateSchema) {
	os.RemoveAll("./generated/")

	// if the structure changes - we can update this inline
	list := `generated/{{ .Project }}/environments/overlays/cicd,generated/{{ .Project }}/environments/overlays/dev/argo,generated/{{ .Project }}/environments/overlays/dev/namespace,generated/{{ .Project }}/environments/overlays/dev/patches,generated/{{ .Project }}/environments/overlays/uat/argo,generated/{{ .Project }}/environments/overlays/uat/namespace,generated/{{ .Project }}/environments/overlays/uat/patches,generated/{{ .Project }}/environments/overlays/prd/argo,generated/{{ .Project }}/environments/overlays/prd/namespace,generated/{{ .Project }}/environments/overlays/prd/patches,generated/{{ .Project }}/environments/overlays/tools,{{range .Items}}generated/{{ .Project }}/manifests/apps/{{ .Application }}/base,{{end}}generated/{{ .Project }}/manifests/apps/namspace-cicd/base,generated/{{ .Project }}/manifests/apps/rbac/base,{{range .Items}}generated/{{ .Project }}/manifests/tekton/pipelines/{{ .Application }}/base,{{end}}generated/{{ .Project }}/manifests/tekton/resources/base,generated/{{ .Project }}/manifests/tekton/task/base,generated/{{ .Project }}/manifests/tekton/tools/base`

	//parse some content and generate a template
	tmpl := template.New("makedirs")
	tmp, _ := tmpl.Parse(list)
	var tpl bytes.Buffer
	tmp.Execute(&tpl, schema)
	schema.Logger.Debug(fmt.Sprintf("Template : %s", tpl.String()))
	dirs := strings.Split(tpl.String(), ",")
	for _, dir := range dirs {
		d := strings.Trim(dir, " ")
		schema.Logger.Info(fmt.Sprintf("Creating directory : %s", d))
		e := os.MkdirAll(d, os.ModePerm)
		if e != nil {
			schema.Logger.Error(fmt.Sprintf("Creating directory : %s", d))
			break
		}
	}
}

// generateApps- utility function to parse and "generate" application yaml templates
func generateApps(schema *GenerateSchema) error {
	files, err := ioutil.ReadDir("./" + schema.ReadPath)
	if err != nil {
		return err
	}

	for x, _ := range schema.Items {
		for _, f := range files {
			data, err := ioutil.ReadFile("./" + schema.ReadPath + "/" + f.Name())
			if err != nil {
				return err
			}
			//parse some content and generate a template
			var tpl bytes.Buffer
			tmpl := template.New(schema.ReadPath)
			tmp, er := tmpl.Parse(string(data))
			if er != nil {
				return er
			}
			tmp.Execute(&tpl, schema.Items[x])
			var path = ""
			if strings.Contains(schema.ReadPath, "patches") {
				path = "./" + schema.Path + "/patch-" + schema.Items[x].Project + "-" + schema.Items[x].Application + "-" + f.Name()
			} else {
				path = "./" + schema.Path + "/" + schema.Items[x].Application + "/base/" + f.Name()
			}
			schema.Logger.Info(fmt.Sprintf("Creating template : %s", path))
			err = ioutil.WriteFile(path, tpl.Bytes(), 0755)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func parseFiles(schema *GenerateSchema) error {
	schema.Logger.Debug(fmt.Sprintf("Path : %s", schema.Path))
	schema.Logger.Debug(fmt.Sprintf("ReadPath : %s", schema.ReadPath))
	files, err := ioutil.ReadDir("./" + schema.ReadPath)
	if err != nil {
		return err
	}

	for _, f := range files {
		fi, _ := os.Stat("./" + schema.ReadPath + "/" + f.Name())
		mode := fi.Mode()
		schema.Logger.Debug(fmt.Sprintf("File mode: %s", mode))
		if mode.IsRegular() {
			data, err := ioutil.ReadFile("./" + schema.ReadPath + "/" + f.Name())
			if err != nil {
				return err
			}
			//parse some content and generate a template
			var tpl bytes.Buffer
			tmpl := template.New(schema.ReadPath)
			tmp, er := tmpl.Parse(string(data))
			if er != nil {
				schema.Logger.Error(fmt.Sprintf("Parsing file : %v", er))
				return er
			}
			tmp.Execute(&tpl, schema)
			schema.Logger.Info(fmt.Sprintf("Creating template : %s", "./"+schema.Path+"/"+f.Name()))
			err = ioutil.WriteFile("./"+schema.Path+"/"+f.Name(), tpl.Bytes(), 0755)
			if err != nil {
				return err
			}
		} else {
			schema.Logger.Debug(fmt.Sprintf("File mode is a directory : %s", f.Name()))
		}
	}
	return nil
}

func injectData(schema *GenerateSchema, env string) *GenerateSchema {
	// we need to inject project data for each application (needed for the template)
	for x, _ := range schema.Items {
		schema.Logger.Debug(fmt.Sprintf("Application  %s", schema.Items[x].Application))
		schema.Logger.Debug(fmt.Sprintf("Repo  %s", schema.Items[x].Repo))
		schema.Items[x].Project = schema.Project
		schema.Items[x].Env = env
	}
	return schema
}

func init() {
	flag.StringVar(&config, "c", "", "Use config file")
	flag.StringVar(&stepsConfig, "s", "", "Use steps file")
}

func main() {
	var schema *GenerateSchema
	var steps *Steps
	var err error

	flag.Parse()
	if config == "" || stepsConfig == "" {
		flag.Usage()
		os.Exit(1)
	}

	logger := &simple.Logger{Level: "info"}
	data, err := ioutil.ReadFile("config.json")
	if err != nil {
		logger.Error(fmt.Sprintf("Reading config.json  %v", err))
		os.Exit(1)
	}
	err = json.Unmarshal(data, &schema)
	if err != nil {
		logger.Error(fmt.Sprintf("Unmarshalling generator struct  %v", err))
		os.Exit(1)
	}
	t := time.Now()
	schema.Time = t.Format(time.RFC3339)
	schema.Logger = logger

	us := injectData(schema, "")
	logger.Debug(fmt.Sprintf("Schema  %v", us))

	si, err := ioutil.ReadFile("steps.json")
	if err != nil {
		logger.Error(fmt.Sprintf("Reading steps.json  %v", err))
		os.Exit(1)
	}
	err = json.Unmarshal(si, &steps)
	if err != nil {
		logger.Error(fmt.Sprintf("Unmarshalling steps struct  %v", err))
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("Steps  %v", steps))
	for _, step := range steps.Items {

		switch step.Name {
		case "mkdirs":
			makeDirs(us)
			break
		case "environments-cicd":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/cicd"
			schema.ReadPath = "templates/environments/cicd"
			err = parseFiles(us)
			break
		case "environments-tools":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/tools"
			schema.ReadPath = "templates/environments/tools"
			err = parseFiles(us)
			break
		case "environments-dev":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev"
			schema.ReadPath = "templates/environments/env"
			schema.Env = "dev"
			ns := injectData(schema, "dev")
			err = parseFiles(ns)
			break
		case "environments-dev-argo":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev/argo"
			schema.ReadPath = "templates/environments/env/argo"
			schema.Env = "dev"
			ns := injectData(schema, "dev")
			err = parseFiles(ns)
			break
		case "environments-dev-namespace":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev/namespace"
			schema.ReadPath = "templates/environments/env/namespace"
			schema.Env = "dev"
			ns := injectData(schema, "dev")
			err = parseFiles(ns)
			break
		case "environments-dev-patches":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev/patches"
			schema.ReadPath = "templates/environments/env/patches"
			schema.Env = "dev"
			ns := injectData(schema, "dev")
			err = generateApps(ns)
			break
		case "environments-uat":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat"
			schema.ReadPath = "templates/environments/env"
			schema.Env = "uat"
			ns := injectData(schema, "uat")
			err = parseFiles(ns)
			break
		case "environments-uat-argo":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat/argo"
			schema.ReadPath = "templates/environments/env/argo"
			schema.Env = "uat"
			ns := injectData(schema, "uat")
			err = parseFiles(ns)
			break
		case "environments-uat-namespace":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat/namespace"
			schema.ReadPath = "templates/environments/env/namespace"
			schema.Env = "uat"
			ns := injectData(schema, "uat")
			err = parseFiles(ns)
			break
		case "environments-uat-patches":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat/patches"
			schema.ReadPath = "templates/environments/env/patches"
			schema.Env = "uat"
			ns := injectData(schema, "uat")
			err = generateApps(ns)
			break
		case "environments-prd":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd"
			schema.ReadPath = "templates/environments/env"
			schema.Env = "prd"
			ns := injectData(schema, "prd")
			err = parseFiles(ns)
			break
		case "environments-prd-argo":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd/argo"
			schema.ReadPath = "templates/environments/env/argo"
			schema.Env = "prd"
			ns := injectData(schema, "prd")
			err = parseFiles(ns)
			break
		case "environments-prd-namespace":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd/namespace"
			schema.ReadPath = "templates/environments/env/namespace"
			schema.Env = "prd"
			ns := injectData(schema, "prd")
			err = parseFiles(ns)
			break
		case "environments-prd-patches":
			schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd/patches"
			schema.ReadPath = "templates/environments/env/patches"
			schema.Env = "prd"
			ns := injectData(schema, "prd")
			err = generateApps(ns)
			break
		case "cicd":
			schema.Path = "./generated/" + schema.Project + "/manifests/apps/namespace-cicd/base"
			schema.ReadPath = "templates/namespace-cicd"
			break
		case "apps":
			schema.Path = "./generated/" + schema.Project + "/manifests/apps/"
			schema.ReadPath = "templates/app"
			err = parseFiles(schema)
			break
		case "rbac":
			schema.Path = "./generated/" + schema.Project + "/manifests/apps/rbac/base"
			schema.ReadPath = "templates/rbac"
			err = parseFiles(schema)
			break
		case "task":
			schema.Path = "./generated/" + schema.Project + "/manifests/tekton/task/base"
			schema.ReadPath = "templates/task"
			err = parseFiles(schema)
			break
		case "tools":
			schema.Path = "./generated/" + schema.Project + "/manifests/tekton/tools/base"
			schema.ReadPath = "templates/tools"
			err = parseFiles(schema)
			break
		case "resources":
			schema.Path = "./generated/" + schema.Project + "/manifests/tekton/resources/base"
			schema.ReadPath = "templates/resources"
			err = parseFiles(schema)
			break
		}

		if err != nil {
			logger.Error(fmt.Sprintf("Parsing files for : %s ", step.Name))
			os.Exit(1)
		}
	}
	os.Exit(0)
}
