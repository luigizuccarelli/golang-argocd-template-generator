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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/microlib/simple"
	"gopkg.in/yaml.v2"
)

type Deployment struct {
	APIVersion string `yaml:"apiVersion"`
	Kind       string `yaml:"kind"`
	Metadata   struct {
		Labels struct {
			App  string `yaml:"app,omitempty"`
			Name string `yaml:"name,omitempty"`
		} `yaml:"labels"`
		Name      string `yaml:"name"`
		Namespace string `yaml:"namespace,omitempty"`
	} `yaml:"metadata"`
	Spec struct {
		Replicas             int `yaml:"replicas"`
		RevisionHistoryLimit int `yaml:"revisionHistoryLimit,omitempty"`
		Selector             struct {
			MatchLabels struct {
				AppKubernetes string `yaml:"app.kubernetes.io"`
			} `yaml:"matchLabels"`
			Name string `yaml:"name,omitempty"`
		} `yaml:"selector"`
		Strategy struct {
			ActiveDeadlineSeconds int `yaml:"activeDeadlineSeconds"`
			Resources             struct {
			} `yaml:"resources"`
			RollingParams struct {
				IntervalSeconds     int    `yaml:"intervalSeconds"`
				MaxSurge            string `yaml:"maxSurge"`
				MaxUnavailable      string `yaml:"maxUnavailable"`
				TimeoutSeconds      int    `yaml:"timeoutSeconds"`
				UpdatePeriodSeconds int    `yaml:"updatePeriodSeconds"`
			} `yaml:"rollingParams"`
			Type string `yaml:"type"`
		} `yaml:"strategy,omitempty"`
		Template struct {
			Metadata struct {
				Annotations struct {
					OpenshiftIoGeneratedBy string `yaml:"openshift.io/generated-by"`
				} `yaml:"annotations,omitempty"`
				CreationTimestamp interface{} `yaml:"creationTimestamp,omitempty"`
				Labels            struct {
					AppKubernetes string `yaml:"app.kubernetes.io"`
					App           string `yaml:"app,omitempty"`
					Name          string `yaml:"name,omitempty"`
				} `yaml:"labels"`
			} `yaml:"metadata"`
			Spec struct {
				Containers []struct {
					Env []struct {
						Name      string `yaml:"name"`
						Value     string `yaml:"value,omitempty"`
						ValueFrom struct {
							SecretKeyRef struct {
								Key  string `yaml:"key"`
								Name string `yaml:"name"`
							} `yaml:"secretKeyRef"`
						} `yaml:"valueFrom,omitempty"`
					} `yaml:"env"`
					Image           string `yaml:"image"`
					ImagePullPolicy string `yaml:"imagePullPolicy,omitempty"`
					LivenessProbe   struct {
						FailureThreshold int `yaml:"failureThreshold"`
						HTTPGet          struct {
							Path   string `yaml:"path"`
							Port   int    `yaml:"port"`
							Scheme string `yaml:"scheme"`
						} `yaml:"httpGet"`
						InitialDelaySeconds int `yaml:"initialDelaySeconds"`
						PeriodSeconds       int `yaml:"periodSeconds"`
						SuccessThreshold    int `yaml:"successThreshold"`
						TimeoutSeconds      int `yaml:"timeoutSeconds"`
					} `yaml:"livenessProbe"`
					Name  string `yaml:"name"`
					Ports []struct {
						ContainerPort int    `yaml:"containerPort"`
						Protocol      string `yaml:"protocol"`
					} `yaml:"ports"`
					ReadinessProbe struct {
						FailureThreshold int `yaml:"failureThreshold"`
						HTTPGet          struct {
							Path   string `yaml:"path"`
							Port   int    `yaml:"port"`
							Scheme string `yaml:"scheme"`
						} `yaml:"httpGet"`
						PeriodSeconds    int `yaml:"periodSeconds"`
						SuccessThreshold int `yaml:"successThreshold"`
						TimeoutSeconds   int `yaml:"timeoutSeconds"`
					} `yaml:"readinessProbe"`
					Resources struct {
						Limits struct {
							CPU    string `yaml:"cpu"`
							Memory string `yaml:"memory"`
						} `yaml:"limits"`
						Requests struct {
							CPU    string `yaml:"cpu"`
							Memory string `yaml:"memory"`
						} `yaml:"requests"`
					} `yaml:"resources"`
					TerminationMessagePath   string `yaml:"terminationMessagePath,omitempty"`
					TerminationMessagePolicy string `yaml:"terminationMessagePolicy,omitempty"`
				} `yaml:"containers"`
				DNSPolicy       string `yaml:"dnsPolicy,omitempty"`
				RestartPolicy   string `yaml:"restartPolicy,omitempty"`
				SchedulerName   string `yaml:"schedulerName,omitempty"`
				SecurityContext struct {
				} `yaml:"securityContext"`
				TerminationGracePeriodSeconds int `yaml:"terminationGracePeriodSeconds,omitempty"`
			} `yaml:"spec"`
		} `yaml:"template"`
		Test     bool `yaml:"test,omitempty"`
		Triggers []struct {
			Type string `yaml:"type"`
		} `yaml:"triggers,omitempty"`
	} `yaml:"spec"`
}

type DeploymentEnvars struct {
	Spec struct {
		Template struct {
			Spec struct {
				Containers []struct {
					Env []struct {
						Name      string `yaml:"name"`
						Value     string `yaml:"value,omitempty"`
						ValueFrom struct {
							SecretKeyRef struct {
								Key  string `yaml:"key"`
								Name string `yaml:"name"`
							} `yaml:"secretKeyRef"`
						} `yaml:"valueFrom,omitempty"`
					} `yaml:"env"`
					Image string `yaml:"image"`
				} `yaml:"containers"`
			} `yaml:"spec"`
		} `yaml:"template"`
	}
}

type GenerateSchema struct {
	Project   string
	Namespace string
	Items     []AppDetails
	Path      string
	ReadPath  string
	Env       string
	Time      string
	Logger    *simple.Logger
}

type AppDetails struct {
	Application string
	Namespace   string
	Port        string
	Repo        string
	Env         string
}

func main() {

	var schema *GenerateSchema
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

	var override *Deployment
	var master *DeploymentEnvars
	for _, items := range schema.Items {
		bs, err := ioutil.ReadFile("generated/" + schema.Project + "/manifests/apps/dev/" + items.Application + "/base/deployment.yaml")
		if err != nil {
			panic(err)
		}
		if err := yaml.Unmarshal(bs, &override); err != nil {
			panic(err)
		}
		bs, err = ioutil.ReadFile("current/" + schema.Project + "/" + items.Application + "/deployment.yaml")
		if err != nil {
			panic(err)
		}
		if err := yaml.Unmarshal(bs, &master); err != nil {
			panic(err)
		}

		// ovverride the envars
		override.Spec.Template.Spec.Containers[0].Env = master.Spec.Template.Spec.Containers[0].Env
		// ovverride the image
		override.Spec.Template.Spec.Containers[0].Image = master.Spec.Template.Spec.Containers[0].Image

		bs, err = yaml.Marshal(override)
		if err != nil {
			panic(err)
		}
		if err := ioutil.WriteFile("generated/myportfolio/manifests/apps/dev/"+items.Application+"/base/deployment.yaml", bs, 0755); err != nil {
			panic(err)
		}
	}
}
