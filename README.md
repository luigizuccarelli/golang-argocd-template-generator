# ArgoCD-Tekton - Template Generator

## Intro

This tool will generate all the necessary yaml files and folders for a project with multiple microservices.

It is meant to do the basic scaffolding and inject the relevant data as needed. 

The are a couple of caveats

- The files generated cannot be *back engineered*, i.e if changes are made to the generated files, and the genrator app is re-run, it will override all changes made.
- Files that contain ENVARS need to be updated manually, it really is extremely difficult to create custom envars per project per microservice.

## CICD folder structure

The folder structure is as follows :

```bash
./generated
      |
      --- <project-name>
                |
                --- environments
                |       |
                |       --- overlays
                |               |
                |               --- cicd
                |               |     |
                |               |     --- kustermization.yaml
                |               |
                |               --- dev
                |               |     |
                |               |     |
                |               |     --- namespace
                |               |     |     |
                |               |     |     --- limit-range.yaml
                |               |     |         namespace.yaml
                |               |     |         resource-quota.yaml
                |               |     |         servis-account-role-binding.yaml
                |               |     |
                |               |     --- kustermization.yaml
                |               |
                |               --- uat (same structure as dev)
                |               |
                |               --- prd (same structure as dev)
                |
                --- manifests
                      |
                      --- apps
                      |     |
                      |     --- microservice-A
                      |     |         |
                      |     |         --- base
                      |     |               |
                      |     |               --- deployment.yaml
                      |     |                   image-pull-secret.yaml
                      |     |                   route.yaml
                      |     |                   service.yaml
                      |     |                   service-account.yaml
                      |     |                   kustermization.yaml
                      |     |
                      |     --- microservice-B (same as microservice-A)
                      |     .
                      |     .
                      |     --- microservice-N (same as microservice-A)
                      |     |
                      |     --- namespace-cicd
                      |     |         |
                      |     |         --- base
                      |     |               |
                      |     |               --- limit-range.yaml
                      |     |                   namespace.yaml
                      |     |                   resource-quota.yaml
                      |     |                   kustermization.yaml
                      |     |
                      |     --- rbac
                      |           |
                      |           --- base
                      |                 |
                      |                 --- (list of user-rbac.yaml files)
                      |
                      --- tekton
                            |
                            --- pipelines
                            |       |
                            |       --- base
                            |             |
                            |             --- microservice-A-pipeline
                            |             |       |    
                            |             |       --- (list of pipeline files)
                            |             .
                            |             .
                            |             --- microservice-N-pipeline
                            |                     |    
                            |                     --- (list of pipeline files)
                            |
                            --- resources
                            |     |
                            |     --- base
                            |           |
                            |           --- git-http-seceret.yaml
                            |               notify-slack-webhook-seceret.yaml
                            |               pipeline-resource-git-http.yaml
                            |               pipeline-resource-git-infra-http.yaml
                            |               pipeline-role.yaml
                            |               smtp-auth.yaml
                            |               kustermization.yaml
                            |
                            --- tasks
                            |     |
                            |     --- base
                            |           |
                            |           --- patch-image.yaml
                            |               promote-image.yaml
                            |               veracode-scanner.yaml
                            |               kustermization.yaml
                            |
                            --- tools
                                  |
                                  --- base
                                        |
                                        --- argocd-appproject.yaml
                                            argocd-apps-cicd.yaml
                                            argocd-apps-dev.yaml
                                            argocd-apps-uat.yaml
                                            argocd-apps-prd.yaml
```
            
## Usage

### Config

Create a simple json file (config.json) - see the example-config.json file in this repo

So basically we are setting up the project name, defining the high level repo's, 
and telling the generator what microservices will be deployed in this app

Don't worry if you don't have all the microservices defined - 
its fairly easy to add to the current project.

**N.B** Remember that if you re-generate the templates any changes made 
in the genertated folder will be lots - so maybe keep a copy (backup) before re-generating

## Steps Config

Use the example-steps.json file as is (you can create your own but leave this file intact its used to generate all yaml objects)

## Build  
Execute the following command in the directory where you have cloned this project
(This assumes you have the golang sdk installed).

```bash
# for mac
env GOOS=darwin GOARCH=amd64 go build -o /build/generate

# for linux
env GOOS=linux GOARCH=amd64 go build -o /build/generate
```

## Execute  
Execute the following command once you have built the executable

```bash
./build/generate -c config.json -s steps.json -l debug

```

All the templates will be generated in the **generated** folder

## Customization

**NB**

You can change any *generated* file as you see fit. As mentioned before if you re-generate all changes will be lost. So backup changes before re-generating

### Basic customization

The main files to customize will be

- ./generated/<project>/manifests/tekton/pipelines/<application>/base/pipeline-dev.yaml (pipeline flow, programming language base image etc)

There is another utility called merge-yaml.go, the generator will create a directory called current/<app> (for each app)
Copy any current deployment yaml files that you have i,e 

```bash
oc get deployment xyz -o yaml > current<app-name>/deployment.yaml
```

### Optional

The merge function will take what is in the folder (current) and iterate through each yaml file and merge it with the generated deploy.yaml

