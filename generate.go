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
	loglevel    string
)

type GenerateSchema struct {
	Project   string
	Namespace string
	Items     []AppDetails
	Path      string
	ReadPath  string
	Env       string
	Time      string
	Version   string
	Logger    *simple.Logger
}

type AppDetails struct {
	Application string
	Namespace   string
	Version     string
	Port        string
	Repo        string
	Replicas    string
	Mem         string
	Cpu         string
	Env         string
}

type Steps struct {
	Project string
	Items   []StepItems
}

type StepItems struct {
	Name string
	Skip bool
}

func makeDirs(schema *GenerateSchema) {
	os.RemoveAll("./generated/")

	// if the structure changes - we can update this inline
	list := `generated/{{ .Project }}/environments/overlays/dev,generated/{{ .Project }}/environments/overlays/dev/namespace,generated/{{ .Project }}/environments/overlays/dev/patches,generated/{{ .Project }}/environments/overlays/uat,generated/{{ .Project }}/environments/overlays/uat/namespace,generated/{{ .Project }}/environments/overlays/uat/patches,generated/{{ .Project }}/environments/overlays/prd,generated/{{ .Project }}/environments/overlays/prd/namespace,generated/{{ .Project }}/environments/overlays/prd/patches`

	//parse some content and generate a template
	tmpl := template.New("makedirs")
	//parse some content and generate a template
	tmp, _ := tmpl.Parse(list)
	var tpl bytes.Buffer
	tmp.Execute(&tpl, schema)
	schema.Logger.Debug(fmt.Sprintf("Template : %s", tpl.String()))
	dirs := strings.Split(tpl.String(), ",")
	for _, dir := range dirs {
		d := strings.Trim(dir, " ")
		e := os.MkdirAll(d, os.ModePerm)
		if e != nil {
			schema.Logger.Error(fmt.Sprintf("Creating directory : %s", d))
			break
		}
		schema.Logger.Debug(fmt.Sprintf("Created directory : %s", d))
	}

	// generate the app dirs
	for _, item := range schema.Items {
		d := "generated/" + schema.Project + "/manifests/apps/" + item.Application + "/base"
		e := os.MkdirAll(d, os.ModePerm)
		if e != nil {
			schema.Logger.Error(fmt.Sprintf("Creating directory : %s", d))
			break
		}
		if _, err := os.Stat("current"); os.IsNotExist(err) {
			d = "current/" + schema.Project + "/" + item.Application
			e = os.MkdirAll(d, os.ModePerm)
			if e != nil {
				schema.Logger.Error(fmt.Sprintf("Creating directory : %s", d))
				break
			}
		}
		schema.Logger.Debug(fmt.Sprintf("Created app directory : %s", d))
	}
	// and lastly for rbac
	d := "generated/" + schema.Project + "/manifests/apps/rbac/base"
	e := os.MkdirAll(d, os.ModePerm)
	if e != nil {
		schema.Logger.Error(fmt.Sprintf("Creating directory : %s", d))
		panic(e)
	}
	schema.Logger.Debug(fmt.Sprintf("Created app directory : %s", d))

}

// generateApps- utility function to parse and "generate" application yaml templates
func generateApps(mode string, schema *GenerateSchema) error {

	files, err := ioutil.ReadDir("./" + schema.ReadPath)
	if err != nil {
		return err
	}
	for _, f := range files {
		for x, _ := range schema.Items {
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
			if mode == "apps" {
				path := schema.Path + "/" + schema.Items[x].Application + "/base/" + f.Name()
				err = ioutil.WriteFile(path, tpl.Bytes(), 0755)
				if err != nil {
					return err
				}
				schema.Logger.Debug(fmt.Sprintf("Created template : %s", path))
			}
			if mode == "patches" {
				path := schema.Path + "/patch-" + schema.Items[x].Application + "-" + f.Name()
				err = ioutil.WriteFile(path, tpl.Bytes(), 0755)
				if err != nil {
					return err
				}
				schema.Logger.Debug(fmt.Sprintf("Created template : %s", path))
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
			err = ioutil.WriteFile("./"+schema.Path+"/"+f.Name(), tpl.Bytes(), 0755)
			if err != nil {
				return err
			}
			schema.Logger.Debug(fmt.Sprintf("Created template : %s", schema.Path+"/"+f.Name()))
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
		schema.Items[x].Env = env
		schema.Items[x].Namespace = schema.Namespace
	}
	return schema
}

func init() {
	flag.StringVar(&loglevel, "l", "info", "Set log level [info,debug,trace]")
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

	logger := &simple.Logger{Level: loglevel}
	data, err := ioutil.ReadFile(config)
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

	logger.Trace(fmt.Sprintf("Schema  %v", schema))

	si, err := ioutil.ReadFile(stepsConfig)
	if err != nil {
		logger.Error(fmt.Sprintf("Reading steps.json  %v", err))
		os.Exit(1)
	}
	err = json.Unmarshal(si, &steps)
	if err != nil {
		logger.Error(fmt.Sprintf("Unmarshalling steps struct  %v", err))
		os.Exit(1)
	}

	logger.Debug(fmt.Sprintf("Steps  %v", steps))
	for _, step := range steps.Items {

		switch step.Name {
		case "mkdirs":
			if !step.Skip {
				makeDirs(schema)
			}
			break
		case "environments-dev":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev"
				schema.ReadPath = "templates/env"
				schema.Env = "dev"
				us := injectData(schema, "dev")
				err = parseFiles(us)
			}
			break
		case "environments-dev-namespace":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev/namespace"
				schema.ReadPath = "templates/namespace"
				schema.Env = "dev"
				us := injectData(schema, "dev")
				err = parseFiles(us)
			}
			break
		case "environments-uat":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat"
				schema.ReadPath = "templates/env"
				schema.Env = "uat"
				us := injectData(schema, "uat")
				err = parseFiles(us)
			}
			break
		case "environments-uat-namespace":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat/namespace"
				schema.ReadPath = "templates/namespace"
				schema.Env = "uat"
				us := injectData(schema, "uat")
				err = parseFiles(us)
			}
			break
		case "environments-prd":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd"
				schema.ReadPath = "templates/env"
				schema.Env = "prd"
				us := injectData(schema, "prd")
				err = parseFiles(us)
			}
			break
		case "environments-prd-namespace":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd/namespace"
				schema.ReadPath = "templates/namespace"
				schema.Env = "prd"
				us := injectData(schema, "prd")
				err = parseFiles(us)
			}
			break
		case "apps":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/manifests/apps"
				schema.ReadPath = "templates/app"
				err = generateApps("apps", schema)
			}
			break
		case "rbac":
			if !step.Skip {
				schema.Path = "./generated/" + schema.Project + "/manifests/apps/rbac/base"
				schema.ReadPath = "templates/rbac"
				err = parseFiles(schema)
			}
			break
		case "patches":
			if !step.Skip {
				us := injectData(schema, "all")
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/dev/patches"
				schema.ReadPath = "templates/patches"
				err = generateApps("patches", us)
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/uat/patches"
				schema.ReadPath = "templates/patches"
				err = generateApps("patches", us)
				schema.Path = "./generated/" + schema.Project + "/environments/overlays/prd/patches"
				schema.ReadPath = "templates/patches"
				err = generateApps("patches", us)
			}
			break
		}

		if err != nil {
			logger.Error(fmt.Sprintf("Parsing files for : %s ", step.Name))
			logger.Error(fmt.Sprintf("Error details  : %v ", err))
			os.Exit(1)
		}
	}
	fmt.Println("")
	logger.Info(fmt.Sprintf("All files can be found in the folder 'generated'"))
	fmt.Println("")
	logger.Warn(fmt.Sprintf("To complete the process copy your manually created patch-<application>-env.yaml files for each environment (dev,uat,prd) to the folders :"))
	logger.Warn(fmt.Sprintf("generated/<projects>/environments/<env>/patches"))
	logger.Warn(fmt.Sprintf("Also add in other resources (i.e secrets, configMaps etc and update the kustomization.yaml file)"))
	logger.Warn(fmt.Sprintf("Finally push your changes to the argocd-repo"))
	logger.Warn(fmt.Sprintf("Remember if you re-generate all changes will be lost. It's recommended to backup your changes"))
	logger.Warn(fmt.Sprintf("Hope this utility was useful, have fun with argocd and tekton :) - LMZ 2020"))
	os.Exit(0)
}
